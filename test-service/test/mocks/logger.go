package mocks

import (
	"github.com/stretchr/testify/mock"
)

// MockLogger is a mock implementation of logger.Logger
type MockLogger struct {
	mock.Mock
}

// Info mocks the Info method
func (m *MockLogger) Info(msg string, keysAndValues ...interface{}) {
	args := []interface{}{msg}
	args = append(args, keysAndValues...)
	m.Called(args...)
}

// Error mocks the Error method
func (m *MockLogger) Error(msg string, keysAndValues ...interface{}) {
	args := []interface{}{msg}
	args = append(args, keysAndValues...)
	m.Called(args...)
}

// Debug mocks the Debug method
func (m *MockLogger) Debug(msg string, keysAndValues ...interface{}) {
	args := []interface{}{msg}
	args = append(args, keysAndValues...)
	m.Called(args...)
}

// Warn mocks the Warn method
func (m *MockLogger) Warn(msg string, keysAndValues ...interface{}) {
	args := []interface{}{msg}
	args = append(args, keysAndValues...)
	m.Called(args...)
}

// Fatal mocks the Fatal method
func (m *MockLogger) Fatal(msg string, keysAndValues ...interface{}) {
	args := []interface{}{msg}
	args = append(args, keysAndValues...)
	m.Called(args...)
}

// With mocks the With method
func (m *MockLogger) With(keysAndValues ...interface{}) *MockLogger {
	m.Called(keysAndValues...)
	return m
}
