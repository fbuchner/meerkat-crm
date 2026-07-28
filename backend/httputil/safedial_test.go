package httputil

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSafeDialContextRejectsNonPublicAddresses(t *testing.T) {
	errUnreachable := errors.New("unreachable")
	errPrivate := errors.New("private")
	dial := SafeDialContext(errUnreachable, errPrivate)

	// Literal IPs skip DNS entirely, so these exercise the classification path
	// directly and stay offline.
	for _, addr := range []string{
		"127.0.0.1:80",
		"10.0.0.1:80",
		"169.254.169.254:80", // cloud metadata
		"100.64.0.1:80",      // CGNAT
		"[::1]:80",
	} {
		t.Run(addr, func(t *testing.T) {
			conn, err := dial(context.Background(), "tcp", addr)
			assert.Nil(t, conn)
			assert.ErrorIs(t, err, errPrivate)
		})
	}
}

func TestSafeDialContextFailsClosedOnResolutionFailure(t *testing.T) {
	errUnreachable := errors.New("unreachable")
	errPrivate := errors.New("private")
	dial := SafeDialContext(errUnreachable, errPrivate)

	// .invalid is reserved by RFC 2606 and never resolves.
	conn, err := dial(context.Background(), "tcp", "nonexistent.invalid:80")
	assert.Nil(t, conn)
	assert.ErrorIs(t, err, errUnreachable)
}

func TestSafeDialContextRejectsMalformedAddress(t *testing.T) {
	dial := SafeDialContext(errors.New("unreachable"), errors.New("private"))

	conn, err := dial(context.Background(), "tcp", "missing-port")
	assert.Nil(t, conn)
	assert.Error(t, err)
}
