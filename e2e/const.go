package e2e

import (
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"time"
)

const (
	retryInterval        = time.Second * 5
	timeout              = time.Second * 300
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 60
	operatorName         = "sample-mysql-operator"
)

var cleanupOptions = framework.CleanupOptions{
	Timeout:       cleanupTimeout,
	RetryInterval: cleanupRetryInterval,
}
