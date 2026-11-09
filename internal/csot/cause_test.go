// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package csot

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/require"
)

func TestWithNamedTimeout(t *testing.T) {
	t.Parallel()

	t.Run("own deadline expires", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := WithNamedTimeout(context.Background(), time.Nanosecond, "connectTimeoutMS")
		defer cancel()
		<-ctx.Done()

		cause := context.Cause(ctx)
		assert.ErrorIs(t, cause, context.DeadlineExceeded)

		var tc *TimeoutCause
		require.ErrorAs(t, cause, &tc)
		assert.Equal(t, "connectTimeoutMS", tc.Name)
		assert.Equal(t, "connectTimeoutMS of 1ns elapsed", cause.Error())
	})

	t.Run("parent cause is retained", func(t *testing.T) {
		t.Parallel()

		userCause := errors.New("request budget exhausted")
		parent, cancelParent := context.WithTimeoutCause(context.Background(), time.Nanosecond, userCause)
		defer cancelParent()

		ctx, cancel := WithNamedTimeout(parent, time.Hour, "timeoutMS")
		defer cancel()
		<-ctx.Done()

		assert.ErrorIs(t, context.Cause(ctx), userCause)
	})
}

func TestContextError(t *testing.T) {
	t.Parallel()

	t.Run("not done", func(t *testing.T) {
		t.Parallel()

		assert.NoError(t, ContextError(context.Background()))
	})

	t.Run("no cause", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		assert.Equal(t, context.Canceled, ContextError(ctx))
	})

	t.Run("named timeout", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := WithNamedTimeout(context.Background(), time.Nanosecond, "serverSelectionTimeoutMS")
		defer cancel()
		<-ctx.Done()

		err := ContextError(ctx)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Equal(t, "context deadline exceeded: serverSelectionTimeoutMS of 1ns elapsed", err.Error())
	})

	t.Run("cause that does not wrap ctx.Err()", func(t *testing.T) {
		t.Parallel()

		userCause := errors.New("request budget exhausted")
		ctx, cancel := context.WithTimeoutCause(context.Background(), time.Nanosecond, userCause)
		defer cancel()
		<-ctx.Done()

		err := ContextError(ctx)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.ErrorIs(t, err, userCause)
		assert.Equal(t, "context deadline exceeded: request budget exhausted", err.Error())
	})
}

func TestWrapCause(t *testing.T) {
	t.Parallel()

	netErr := errors.New("dial tcp 192.0.2.1:27017: i/o timeout")

	t.Run("not done", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, netErr, WrapCause(context.Background(), netErr))
	})

	t.Run("no cause", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		assert.Equal(t, netErr, WrapCause(ctx, netErr))
	})

	t.Run("named timeout", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := WithNamedTimeout(context.Background(), time.Nanosecond, "connectTimeoutMS")
		defer cancel()
		<-ctx.Done()

		err := WrapCause(ctx, netErr)
		assert.ErrorIs(t, err, netErr)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Equal(t, "dial tcp 192.0.2.1:27017: i/o timeout: connectTimeoutMS of 1ns elapsed", err.Error())
	})

	t.Run("err already contains cause", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := WithNamedTimeout(context.Background(), time.Nanosecond, "connectTimeoutMS")
		defer cancel()
		<-ctx.Done()

		err := ContextError(ctx)
		assert.Equal(t, err, WrapCause(ctx, err))
	})
}
