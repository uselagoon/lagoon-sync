//go:build integration

package synchers

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// lagoonMariaDBImage is the Lagoon base image used in production Lagoon environments.
// It pre-configures MARIADB_USER=drupal, MARIADB_PASSWORD=drupal, MARIADB_DATABASE=drupal.
const lagoonMariaDBImage = "uselagoon/mariadb-10.11-drupal:latest"

// TestMariadbSyncer_GetRemoteCommand_Integration spins up a real Lagoon MariaDB container,
// renders the commands produced by GetRemoteCommand, executes them inside the container,
// and asserts that the expected dump file is created.
//
// Run with: go test -tags=integration ./synchers/ -run TestMariadbSyncer_GetRemoteCommand_Integration -v
func TestMariadbSyncer_GetRemoteCommand_Integration(t *testing.T) {
	ctx := context.Background()

	// The Lagoon image pre-bakes these credentials; no env overrides needed.
	const (
		dbUser     = "drupal"
		dbPassword = "drupal"
		dbDatabase = "drupal"
		dbPort     = "3306"
	)

	// Use a generic container so we can supply the correct wait strategy for the
	// Lagoon image, which emits a different ready log line than the upstream image.
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        lagoonMariaDBImage,
			ExposedPorts: []string{"3306/tcp"},
			WaitingFor:   wait.ForLog("port: 3306  Alpine Linux"),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("failed to start %s container: %v", lagoonMariaDBImage, err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("warning: failed to terminate container: %v", err)
		}
	})

	// Build the syncer.
	// Commands execute *inside* the container, so we target 127.0.0.1 on the
	// internal port; the mapped host port is only needed for external clients.
	syncer := &MariadbSyncRoot{
		Config: BaseMariaDbSync{
			DbHostname: "127.0.0.1",
			DbUsername: dbUser,
			DbPassword: dbPassword,
			DbPort:     dbPort,
			DbDatabase: dbDatabase,
		},
	}
	if _, err := syncer.PrepareSyncer(); err != nil {
		t.Fatalf("PrepareSyncer() error: %v", err)
	}

	// Use a non-local environment so GetRemoteCommand uses syncer.Config directly.
	sourceEnv := Environment{
		ProjectName:     "test-project",
		EnvironmentName: "test-env",
	}

	commands := syncer.GetRemoteCommand(sourceEnv)
	if len(commands) != 2 {
		t.Fatalf("expected GetRemoteCommand to return 2 commands, got %d", len(commands))
	}

	// Render and execute each command inside the container.
	for i, syncCmd := range commands {
		cmdStr, err := syncCmd.GetCommand()
		if err != nil {
			t.Fatalf("command[%d] GetCommand() error: %v", i, err)
		}

		t.Logf("executing command[%d]: %s", i, cmdStr)
		exitCode, output, err := container.Exec(ctx, []string{"bash", "-c", cmdStr})
		if err != nil {
			t.Fatalf("command[%d] container.Exec() error: %v", i, err)
		}

		// Drain output so it is available for failure diagnostics.
		var sb strings.Builder
		if _, copyErr := io.Copy(&sb, output); copyErr != nil {
			t.Logf("command[%d] warning: could not read output: %v", i, copyErr)
		}
		if sb.Len() > 0 {
			t.Logf("command[%d] output:\n%s", i, sb.String())
		}

		if exitCode != 0 {
			t.Fatalf("command[%d] exited with code %d.\ncommand: %s\noutput: %s",
				i, exitCode, cmdStr, sb.String())
		}
	}

	// Assert the gzip-compressed dump file exists inside the container.
	transferResource := syncer.GetTransferResource(sourceEnv)
	dumpFile := transferResource.Name
	t.Logf("verifying dump file exists: %s", dumpFile)

	exitCode, _, err := container.Exec(ctx, []string{"ls", dumpFile})
	if err != nil {
		t.Fatalf("ls %s: exec error: %v", dumpFile, err)
	}
	if exitCode != 0 {
		t.Fatalf("dump file not found at %s (ls exited %d)", dumpFile, exitCode)
	}

	t.Logf("SUCCESS: dump file created at %s", dumpFile)
}
