package util

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"
)

func DeSlasher(str string) string {
	return strings.Replace(str, "/", "-", -1)
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

// This function is for view layer. If using elsewhere, consider carefully.
// There are reasons go doesn't behave this way by default.
func SafeSlice(s string, start, end int) string {
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

func RoleArnFromAssumeRoleArn(arn string) (string, error) {
	if !strings.Contains(arn, "assumed-role") {
		return "", fmt.Errorf("no assumed role found in ARN; self assumable role is the only valid type of caller")
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
