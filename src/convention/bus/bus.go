package bus

import (
	"context"
	"fmt"
	"strings"

	"github.com/linecard/self/convention/config"
	"github.com/linecard/self/convention/deployment"
	"github.com/linecard/self/internal/labelgun"
	"github.com/linecard/self/service/event"

	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	dockerTypes "github.com/docker/docker/api/types"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type RegistryService interface {
	InspectByDigest(ctx context.Context, registryId, repository, digest string) (dockerTypes.ImageInspect, error)
}

type EventService interface {
	List(ctx context.Context) ([]event.JoinedRule, error)
	Put(ctx context.Context, bus, rule, expression, function, arn string) error
	Delete(ctx context.Context, bus, rule, function, arn string) error
}

type Meta struct {
	Bus         string
	Rule        string
	Destroy     bool
	Update      bool
	Reason      string
	Convergence string
	Expression  string
}

type Subscription struct {
	event.JoinedRule
	Meta Meta
}

type Services struct {
	Registry RegistryService
	Event    EventService
}

type Convention struct {
	Config  config.Config
	Service Services
}

func FromServices(c config.Config, r RegistryService, e EventService) Convention {
	return Convention{
		Config: c,
		Service: Services{
			Registry: r,
			Event:    e,
		},
	}
}

func (c Convention) Find(ctx context.Context, d deployment.Deployment, bus, rule string) (Subscription, error) {
	subscriptions, err := c.List(ctx, d)
	if err != nil {
		return Subscription{}, err
	}

	for _, subscription := range subscriptions {
		if subscription.Meta.Bus == bus && subscription.Meta.Rule == rule {
			return subscription, nil
		}
	}

	return Subscription{}, fmt.Errorf("subscription not found, bus: %s, rule: %s", bus, rule)
}

func (c Convention) List(ctx context.Context, d deployment.Deployment) ([]Subscription, error) {
	var subscriptions []Subscription
	var update []Subscription
	var delete []Subscription
	var noop []Subscription

	span := trace.SpanFromContext(ctx)
	span.End()
	span.SetName("bus.List")

	definitions, err := c.listDefined(ctx, d)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return []Subscription{}, err
	}

	active, err := c.listEnabled(ctx, d)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return []Subscription{}, err
	}

	// The O(n) of this situation is really bad, but it'll never be slow, and so far it's the clearest expression I could muster.
	for _, activeRule := range active {
		shortName := strings.Replace(*activeRule.Rule.Name, *d.Configuration.FunctionName+"-", "", 1)
		// If the rule is defined and enabled, we need to update it.
		// If the rule is enabled but not defined, we need to delete it.
		if c.containsRule(definitions, *activeRule.Rule.Name) {
			expression, err := c.Config.Template(c.getExpression(definitions, *activeRule.Rule.Name))
			if err != nil {
				span.SetStatus(codes.Error, err.Error())
				return []Subscription{}, err
			}

			activeRule.Meta.Bus = *activeRule.Bus.Name
			activeRule.Meta.Rule = shortName
			activeRule.Meta.Destroy = false
			activeRule.Meta.Update = true
			activeRule.Meta.Reason = "Enabled and Defined"
			activeRule.Meta.Convergence = "Update"
			activeRule.Meta.Expression = expression
			update = append(update, activeRule)
		}

		if !c.containsRule(definitions, *activeRule.Rule.Name) {
			activeRule.Meta.Bus = *activeRule.Bus.Name
			activeRule.Meta.Rule = shortName
			activeRule.Meta.Destroy = true
			activeRule.Meta.Update = false
			activeRule.Meta.Reason = "Enabled and Not Defined"
			activeRule.Meta.Convergence = "Destroy"
			delete = append(delete, activeRule)
		}
	}

	// If the rule is defined but not enabled, we don't do anything.
	for _, definedRule := range definitions {
		shortName := strings.Replace(*definedRule.Rule.Name, *d.Configuration.FunctionName+"-", "", 1)

		if !c.containsRule(active, *definedRule.Rule.Name) {
			expression, err := c.Config.Template(c.getExpression(definitions, *definedRule.Rule.Name))
			if err != nil {
				span.SetStatus(codes.Error, err.Error())
				return []Subscription{}, err
			}

			definedRule.Meta.Bus = *definedRule.Bus.Name
			definedRule.Meta.Rule = shortName
			definedRule.Meta.Destroy = false
			definedRule.Meta.Update = false
			definedRule.Meta.Reason = "Defined and Not Enabled"
			definedRule.Meta.Convergence = "Noop"
			definedRule.Meta.Expression = expression
			noop = append(noop, definedRule)
		}
	}

	subscriptions = append(subscriptions, update...)
	subscriptions = append(subscriptions, delete...)
	subscriptions = append(subscriptions, noop...)

	return subscriptions, nil
}

func (c Convention) Disable(ctx context.Context, d deployment.Deployment, s Subscription) error {
	span := trace.SpanFromContext(ctx)
	span.End()
	span.SetName("bus.Disable")

	err := c.Service.Event.Delete(ctx, *s.Bus.Name, *s.Rule.Name, *d.Configuration.FunctionName, *d.Configuration.FunctionArn)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	return nil
}

func (c Convention) Enable(ctx context.Context, d deployment.Deployment, s Subscription) error {
	return c.Service.Event.Put(ctx, *s.Bus.Name, *s.Rule.Name, s.Meta.Expression, *d.Configuration.FunctionName, *d.Configuration.FunctionArn)
}

func (c Convention) EnableAll(ctx context.Context, d deployment.Deployment) error {
	subscriptions, err := c.List(ctx, d)
	if err != nil {
		return err
	}

	for _, subscription := range subscriptions {
		c.Enable(ctx, d, subscription)
	}

	return nil
}

func (c Convention) DisableAll(ctx context.Context, d deployment.Deployment) error {
	subscriptions, err := c.List(ctx, d)
	if err != nil {
		return err
	}

	for _, subscription := range subscriptions {
		if err := c.Disable(ctx, d, subscription); err != nil {
			return err
		}
	}

	return nil
}

func (c Convention) Converge(ctx context.Context, d deployment.Deployment) error {
	span := trace.SpanFromContext(ctx)
	defer span.End()
	span.SetName("bus.Converge")

	subscriptions, err := c.List(ctx, d)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	for _, subscription := range subscriptions {
		if subscription.Meta.Destroy {
			if err := c.Disable(ctx, d, subscription); err != nil {
				span.SetStatus(codes.Error, err.Error())
				return err
			}
		}

		if subscription.Meta.Update {
			if err := c.Enable(ctx, d, subscription); err != nil {
				span.SetStatus(codes.Error, err.Error())
				return err
			}
		}
	}
	return nil
}

func (c Convention) listDefined(ctx context.Context, d deployment.Deployment) ([]Subscription, error) {
	var subscriptions []Subscription

	r, err := d.FetchRelease(ctx, c.Service.Registry, c.Config.Registry.Id)
	if err != nil {
		return []Subscription{}, err
	}

	busLabels, err := labelgun.DecodeLabels(c.Config.Label.Bus, r.Config.Labels)
	if err != nil {
		return []Subscription{}, err
	}

	for label, value := range busLabels {
		parts := strings.Replace(label, c.Config.Label.Bus, "", 1)
		parts = strings.TrimPrefix(parts, ".")

		bus := strings.Split(parts, ".")[0]
		rule := strings.Split(parts, ".")[1]
		rule = *d.Configuration.FunctionName + "-" + rule

		subscriptions = append(subscriptions, Subscription{
			event.JoinedRule{
				Bus: types.EventBus{
					Name: &bus,
				},
				Rule: types.Rule{
					Name: &rule,
				},
			},
			Meta{
				Expression: value,
			},
		})
	}

	return subscriptions, nil
}

func (c Convention) listEnabled(ctx context.Context, d deployment.Deployment) ([]Subscription, error) {
	var activeSubscriptions []Subscription

	subscriptions, err := c.Service.Event.List(ctx)
	if err != nil {
		return []Subscription{}, err
	}

	for _, channel := range subscriptions {
		if *channel.Target.Arn == *d.Configuration.FunctionArn {
			activeSubscriptions = append(activeSubscriptions, Subscription{channel, Meta{}})
		}
	}

	return activeSubscriptions, nil
}

func (c Convention) containsRule(slice []Subscription, ruleName string) bool {
	for _, item := range slice {
		if *item.Rule.Name == ruleName {
			return true
		}
	}
	return false
}

func (c Convention) getExpression(subscriptions []Subscription, ruleName string) string {
	for _, sub := range subscriptions {
		if *sub.Rule.Name == ruleName {
			return sub.Meta.Expression
		}
	}

	return ""
}
