package bkimage

import (
	"context"
	"fmt"

	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/filesync"
	"github.com/moby/buildkit/session/testutil"
)

const sessionName = "forge"

// Session creates a new session and its dialer.
//
// The localDirs should reference the "context" and "dockerfile" directories for a particular build.
func (c *Client) Session(ctx context.Context, localDirs map[string]string) (*session.Session, session.Dialer, error) {
	// fetch the session manager
	sm, err := c.getSessionManager()
	if err != nil {
		return nil, nil, err
	}

	// create and configure a new session
	sess, err := session.NewSession(ctx, sessionName, "")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create session: %w", err)
	}

	var syncedDirs []filesync.SyncedDir
	for name, dir := range localDirs {
		syncedDirs = append(syncedDirs, filesync.SyncedDir{
			Name: name,
			Dir:  dir,
		})
	}

	sess.Allow(filesync.NewFSSyncProvider(syncedDirs))
	sess.Allow(NewDynamicAuthProvider(c.getHostCredentials()))

	// create a session dialer
	dialer := session.Dialer(testutil.TestStream(testutil.Handler(sm.HandleConn)))

	return sess, dialer, nil
}

// Creates an instance (singleton) of the buildkit session manager and adds it to the client.
func (c *Client) getSessionManager() (*session.Manager, error) {
	if c.sessionManager == nil {
		var err error
		c.sessionManager, err = session.NewManager()
		if err != nil {
			return nil, fmt.Errorf("cannot create session manager: %w", err)
		}
	}

	return c.sessionManager, nil
}
