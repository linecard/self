package util

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"

	"github.com/aws/smithy-go/logging"
	"github.com/rs/zerolog"
)

func DeSlasher(str string) string {
	dashes := strings.Replace(str, "/", "-", -1)
	dashes = strings.TrimSuffix(dashes, "-")
	dashes = strings.TrimPrefix(dashes, "-")
	return dashes
}

func ReSlasher(str string) string {
	return strings.Replace(str, "-", "/", -1)
}

func ShaLike(str string) bool {
	regexExp := regexp.MustCompile(`^[a-f0-9]{40}$`)
	return regexExp.MatchString(str)
}

func RoleNameFromArn(arn string) string {
	return strings.Split(arn, ":role/")[1]
}

func PolicyNameFromArn(arn string) string {
	return strings.Split(arn, ":policy/")[1]
}

func RoleArnFromName(accountId, name string) string {
	return "arn:aws:iam::" + accountId + ":role/" + name
}

func PolicyArnFromName(accountId, name string) string {
	return "arn:aws:iam::" + accountId + ":policy/" + name
}

// For view layer only
func UnsafeSlice(s string, start, end int) string {
	if s == "" {
		return ""
	}
	if end > len(s) {
		end = len(s)
	}
	if start > len(s) {
		return ""
	}
	return s[start:end]
}

func Chomp(s string) string {
	s = strings.TrimLeftFunc(s, unicode.IsSpace)
	s = strings.TrimRightFunc(s, unicode.IsSpace)
	return s
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func InLambda() bool {
	_, inLambda := os.LookupEnv("AWS_LAMBDA_FUNCTION_NAME")
	return inLambda
}

func OtelConfigPresent() bool {
	_, present := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
	return present
}

func RoleArnFromAssumeRoleArn(arn string) (string, error) {
	if !strings.Contains(arn, "assumed-role") {
		return "", fmt.Errorf("assumed-role not found in arn, policy emulation only supports self-assumable roles")
	}

	// Split the ARN to find the assumed-role part
	parts := strings.Split(arn, ":assumed-role/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid ARN format, expecting ':assumed-role/' part")
	}

	// Extract account ID and the role name
	accountID := parts[0][len("arn:aws:sts::"):]
	roleParts := strings.Split(parts[1], "/")
	if len(roleParts) < 2 {
		return "", fmt.Errorf("invalid assumed role ARN format")
	}
	roleName := roleParts[0]

	// Construct the IAM role ARN
	iamArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, roleName)
	return iamArn, nil
}

func SetLogLevel() {
	if level, exists := os.LookupEnv("LOG_LEVEL"); exists {
		level = strings.ToLower(level)
		switch level {
		case "panic":
			zerolog.SetGlobalLevel(zerolog.PanicLevel)
		case "fatal":
			zerolog.SetGlobalLevel(zerolog.FatalLevel)
		case "error":
			zerolog.SetGlobalLevel(zerolog.ErrorLevel)
		case "warn":
			zerolog.SetGlobalLevel(zerolog.WarnLevel)
		case "info":
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		case "debug":
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		case "trace":
			zerolog.SetGlobalLevel(zerolog.TraceLevel)
		default:
			zerolog.SetGlobalLevel(zerolog.WarnLevel)
		}
		return
	}

	zerolog.SetGlobalLevel(zerolog.WarnLevel)
}

// Wraps a zerolog.Logger so its interoperable with Go's standard "log" package

type AwsLogInterface interface {
	// Logf is expected to support the standard fmt package "verbs".
	Logf(classification logging.Classification, format string, v ...interface{})
}

type RetryLogger struct {
	Log *zerolog.Logger
}

func (l *RetryLogger) Logf(classification logging.Classification, format string, v ...interface{}) {
	switch classification {
	case "WARN":
		l.Log.Warn().Msgf(format, v...)
	case "DEBUG":
		if strings.Contains(format, "retrying request") {
			l.Log.Info().Msgf(format, v...)
		} else {
			l.Log.Debug().Msgf(format, v...)
		}
	default:
		l.Log.Error().Msgf(format, v...)
	}
}
