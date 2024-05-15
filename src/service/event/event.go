package event

import (
	"context"
	"errors"
	"strings"

	"github.com/linecard/self/internal/util"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/smithy-go"
)

type EventBridgeClient interface {
	ListEventBuses(ctx context.Context, params *eventbridge.ListEventBusesInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListEventBusesOutput, error)
	ListRules(ctx context.Context, params *eventbridge.ListRulesInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListRulesOutput, error)
	ListTargetsByRule(ctx context.Context, params *eventbridge.ListTargetsByRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListTargetsByRuleOutput, error)
	PutRule(ctx context.Context, params *eventbridge.PutRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.PutRuleOutput, error)
	PutTargets(ctx context.Context, params *eventbridge.PutTargetsInput, optFns ...func(*eventbridge.Options)) (*eventbridge.PutTargetsOutput, error)
	RemoveTargets(ctx context.Context, params *eventbridge.RemoveTargetsInput, optFns ...func(*eventbridge.Options)) (*eventbridge.RemoveTargetsOutput, error)
	DeleteRule(ctx context.Context, params *eventbridge.DeleteRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.DeleteRuleOutput, error)
}

type LambdaClient interface {
	AddPermission(ctx context.Context, params *lambda.AddPermissionInput, optFns ...func(*lambda.Options)) (*lambda.AddPermissionOutput, error)
	RemovePermission(ctx context.Context, params *lambda.RemovePermissionInput, optFns ...func(*lambda.Options)) (*lambda.RemovePermissionOutput, error)
}

type JoinedRule struct {
	Bus    types.EventBus
	Rule   types.Rule
	Target types.Target
}

type Client struct {
	EventBridge EventBridgeClient
	Lambda      LambdaClient
}

type Service struct {
	Client Client
}

func FromClients(eventBridge EventBridgeClient, lambdaClient LambdaClient) Service {
	return Service{
		Client: Client{
			EventBridge: eventBridge,
			Lambda:      lambdaClient,
		},
	}
}

func (s Service) List(ctx context.Context) ([]JoinedRule, error) {
	var results []JoinedRule

	buses, err := s.Client.EventBridge.ListEventBuses(ctx, nil)
	if err != nil {
		return []JoinedRule{}, err
	}

	for _, bus := range buses.EventBuses {
		rules, err := s.Client.EventBridge.ListRules(ctx, &eventbridge.ListRulesInput{
			EventBusName: bus.Name,
		})

		if err != nil {
			return []JoinedRule{}, err
		}

		for _, rule := range rules.Rules {
			ruleTargets, err := s.Client.EventBridge.ListTargetsByRule(ctx, &eventbridge.ListTargetsByRuleInput{
				EventBusName: bus.Name,
				Rule:         rule.Name,
			})

			if err != nil {
				return []JoinedRule{}, err
			}

			for _, target := range ruleTargets.Targets {
				results = append(results, JoinedRule{
					Bus:    bus,
					Rule:   rule,
					Target: target,
				})
			}
		}
	}

	return results, nil
}

func (s Service) Put(ctx context.Context, busName, ruleName, ruleContent, functionName, functionArn string) error {
	var apiErr smithy.APIError

	putRuleInput := eventbridge.PutRuleInput{
		EventBusName: aws.String(busName),
		Name:         aws.String(ruleName),
		State:        types.RuleStateEnabled,
		Description:  aws.String("managed by self"),
	}

	if strings.HasPrefix(ruleContent, "cron(") || strings.HasPrefix(ruleContent, "rate(") {
		scheduleExpression := util.Chomp(ruleContent)
		putRuleInput.ScheduleExpression = aws.String(scheduleExpression)
	} else {
		putRuleInput.EventPattern = aws.String(ruleContent)
	}

	putTargetsInput := eventbridge.PutTargetsInput{
		EventBusName: aws.String(busName),
		Rule:         aws.String(ruleName),
		Targets: []types.Target{
			{
				Id:  aws.String(functionName),
				Arn: aws.String(functionArn),
			},
		},
	}

	putRuleOutput, err := s.Client.EventBridge.PutRule(ctx, &putRuleInput)
	if err != nil {
		return err
	}

	_, err = s.Client.EventBridge.PutTargets(ctx, &putTargetsInput)
	if err != nil {
		return err
	}

	addPermissionsInput := lambda.AddPermissionInput{
		FunctionName: aws.String(functionName),
		StatementId:  aws.String(ruleName),
		Action:       aws.String("lambda:InvokeFunction"),
		Principal:    aws.String("events.amazonaws.com"),
		SourceArn:    aws.String(*putRuleOutput.RuleArn),
	}

	if _, err := s.Client.Lambda.AddPermission(ctx, &addPermissionsInput); err != nil {
		if errors.As(err, &apiErr) {
			switch apiErr.ErrorCode() {
			case "ResourceConflictException":
				break
			default:
				return err
			}
		}
	}

	return nil
}

func (s Service) Delete(ctx context.Context, busName, ruleName, functionName, functionArn string) error {
	var apiErr smithy.APIError

	removeTargetsInput := eventbridge.RemoveTargetsInput{
		EventBusName: aws.String(busName),
		Rule:         aws.String(ruleName),
		Ids:          []string{functionName},
	}

	deleteRuleInput := eventbridge.DeleteRuleInput{
		EventBusName: aws.String(busName),
		Name:         aws.String(ruleName),
	}

	removePermissionInput := lambda.RemovePermissionInput{
		FunctionName: aws.String(functionName),
		StatementId:  aws.String(ruleName),
	}

	if _, err := s.Client.EventBridge.RemoveTargets(ctx, &removeTargetsInput); err != nil {
		if errors.As(err, &apiErr) {
			switch apiErr.ErrorCode() {
			case "ResourceNotFoundException":
				break
			default:
				return err
			}
		}
	}

	if _, err := s.Client.EventBridge.DeleteRule(ctx, &deleteRuleInput); err != nil {
		return err
	}

	if _, err := s.Client.Lambda.RemovePermission(ctx, &removePermissionInput); err != nil {
		if errors.As(err, &apiErr) {
			switch apiErr.ErrorCode() {
			case "ResourceNotFoundException":
				break
			default:
				return err
			}
		}
	}

	return nil
}
