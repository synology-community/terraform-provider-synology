package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type SynoLogger struct {
	ctx context.Context
}

func NewLogger(ctx context.Context) *SynoLogger {
	return &SynoLogger{
		ctx: ctx,
	}
}

func (l *SynoLogger) Error(msg string, keysAndValues ...any) {
	additionalFields, err := l.convertToAdditionalFields(keysAndValues)
	if err != nil {
		tflog.Error(context.Background(), fmt.Sprintf("Error converting keys and values: %v", err))
		return
	}
	tflog.Error(context.Background(), msg, additionalFields)
}

func (l *SynoLogger) Info(msg string, keysAndValues ...any) {
	additionalFields, err := l.convertToAdditionalFields(keysAndValues)
	if err != nil {
		tflog.Error(context.Background(), fmt.Sprintf("Error converting keys and values: %v", err))
		return
	}
	tflog.Info(context.Background(), msg, additionalFields)
}

func (l *SynoLogger) Debug(msg string, keysAndValues ...any) {
	additionalFields, err := l.convertToAdditionalFields(keysAndValues)
	if err != nil {
		tflog.Error(context.Background(), fmt.Sprintf("Error converting keys and values: %v", err))
		return
	}
	tflog.Debug(context.Background(), msg, additionalFields)
}

func (l *SynoLogger) Warn(msg string, keysAndValues ...any) {
	additionalFields, err := l.convertToAdditionalFields(keysAndValues)
	if err != nil {
		tflog.Error(context.Background(), fmt.Sprintf("Error converting keys and values: %v", err))
		return
	}
	tflog.Warn(context.Background(), msg, additionalFields)
}

func (l *SynoLogger) convertToAdditionalFields(keysAndValues []any) (map[string]any, error) {
	additionalFields := make(map[string]any, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 >= len(keysAndValues) {
			return nil, fmt.Errorf("missing value for key %s", keysAndValues[i])
		}

		if key, ok := keysAndValues[i].(string); ok {
			additionalFields[key] = keysAndValues[i+1]
		} else {
			return nil, fmt.Errorf("key %v is not a string", keysAndValues[i])
		}
	}
	return additionalFields, nil
}
