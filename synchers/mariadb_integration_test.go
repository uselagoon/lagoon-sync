//go:build integration

package synchers

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// lagoonMariaDBImage is the Lagoon base image used in production Lagoon environments.
// It pre-configures MARIADB_USER=drupal, MARIADB_PASSWORD=drupal, MARIADB_DATABASE=drupal.
const lagoonMariaDBImage = "uselagoon/mariadb-10.11-drupal:latest"

// startLagoonMariaDB starts a Lagoon MariaDB container and returns it ready for use.
func startLagoonMariaDB(ctx context.Context, t *testing.T) testcontainers.Container {
	t.Helper()
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
	return container
}

// execInContainer runs a shell command inside the container, logs its output, and
// fails the test if the exit code is non-zero.
func execInContainer(ctx context.Context, t *testing.T, container testcontainers.Container, cmd string) {
	t.Helper()
	t.Logf("exec: %s", cmd)
	exitCode, output, err := container.Exec(ctx, []string{"bash", "-c", cmd})
	if err != nil {
		t.Fatalf("container.Exec() error: %v\ncmd: %s", err, cmd)
	}
	var sb strings.Builder
	if _, copyErr := io.Copy(&sb, output); copyErr != nil {
		t.Logf("warning: could not read exec output: %v", copyErr)
	}
	if sb.Len() > 0 {
		t.Logf("output:\n%s", sb.String())
	}
	if exitCode != 0 {
		t.Fatalf("command exited with code %d\ncmd: %s\noutput: %s", exitCode, cmd, sb.String())
	}
}

// execInContainerOutput is like execInContainer but returns stdout/stderr as a string
// instead of failing on non-zero exit — useful for queries where you want to inspect output.
func execInContainerOutput(ctx context.Context, t *testing.T, container testcontainers.Container, cmd string) (int, string) {
	t.Helper()
	exitCode, output, err := container.Exec(ctx, []string{"bash", "-c", cmd})
	if err != nil {
		t.Fatalf("container.Exec() error: %v\ncmd: %s", err, cmd)
	}
	var sb strings.Builder
	if _, copyErr := io.Copy(&sb, output); copyErr != nil {
		t.Logf("warning: could not read exec output: %v", copyErr)
	}
	return exitCode, sb.String()
}

// TestMariadbSyncer_GetRemoteCommand_Integration spins up a real Lagoon MariaDB container,
// renders the commands produced by GetRemoteCommand, executes them inside the container,
// and asserts that the expected dump file is created.
//
// Run with: go test -tags=integration ./synchers/ -run TestMariadbSyncer_GetRemoteCommand_Integration -v
func TestMariadbSyncer_GetRemoteCommand_Integration(t *testing.T) {
	ctx := context.Background()

	container := startLagoonMariaDB(ctx, t)

	syncer := &MariadbSyncRoot{
		Config: BaseMariaDbSync{
			DbHostname: "127.0.0.1",
			DbUsername: "drupal",
			DbPassword: "drupal",
			DbPort:     "3306",
			DbDatabase: "drupal",
		},
	}
	if _, err := syncer.PrepareSyncer(); err != nil {
		t.Fatalf("PrepareSyncer() error: %v", err)
	}

	sourceEnv := Environment{
		ProjectName:     "test-project",
		EnvironmentName: "test-env",
	}

	commands := syncer.GetRemoteCommand(sourceEnv)
	if len(commands) != 2 {
		t.Fatalf("expected GetRemoteCommand to return 2 commands, got %d", len(commands))
	}

	for i, syncCmd := range commands {
		cmdStr, err := syncCmd.GetCommand()
		if err != nil {
			t.Fatalf("command[%d] GetCommand() error: %v", i, err)
		}
		execInContainer(ctx, t, container, cmdStr)
	}

	transferResource := syncer.GetTransferResource(sourceEnv)
	dumpFile := transferResource.Name
	t.Logf("verifying dump file exists: %s", dumpFile)

	exitCode, _ := execInContainerOutput(ctx, t, container, fmt.Sprintf("ls %s", dumpFile))
	if exitCode != 0 {
		t.Fatalf("dump file not found at %s", dumpFile)
	}
	t.Logf("SUCCESS: dump file created at %s", dumpFile)
}

// TestMariadbSyncer_GetLocalCommand_Integration copies a pre-made Umami Drupal database
// fixture (testdata/test_dump.sql.gz) into a Lagoon MariaDB container, runs the commands
// produced by GetLocalCommand to import it, and asserts the data landed correctly by
// checking that the `node` table contains at least one row.
//
// Run with: go test -tags=integration ./synchers/ -run TestMariadbSyncer_GetLocalCommand_Integration -v
func TestMariadbSyncer_GetLocalCommand_Integration(t *testing.T) {
	const fixtureFile = "testdata/test_dump.sql.gz"
	const containerDumpPath = "/tmp/test_dump.sql.gz"

	// Fail early with a clear message if the fixture hasn't been generated yet.
	if _, err := os.Stat(fixtureFile); os.IsNotExist(err) {
		t.Fatalf("fixture not found at %s — generate it with: mysqldump ... drupal | gzip > %s", fixtureFile, fixtureFile)
	}

	ctx := context.Background()
	container := startLagoonMariaDB(ctx, t)

	// Copy the fixture into the container.
	fixtureBytes, err := os.ReadFile(fixtureFile)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", fixtureFile, err)
	}
	// CopyToContainer places files as root-owned. The container runs as a non-root
	// user that can't chmod or unlink root-owned files (which gunzip requires).
	// Work around this by copying to a staging path and then using `cp` inside the
	// container — cp runs as the container user and creates a file it owns.
	const stagingPath = "/tmp/test_dump_staging.sql.gz"
	if err := container.CopyToContainer(ctx, fixtureBytes, stagingPath, 0o644); err != nil {
		t.Fatalf("failed to copy fixture into container: %v", err)
	}
	execInContainer(ctx, t, container, fmt.Sprintf("cp %s %s", stagingPath, containerDumpPath))
	t.Logf("copied %s into container at %s (%d bytes)", fixtureFile, containerDumpPath, len(fixtureBytes))

	syncer := &MariadbSyncRoot{
		Config: BaseMariaDbSync{
			DbHostname: "127.0.0.1",
			DbUsername: "drupal",
			DbPassword: "drupal",
			DbPort:     "3306",
			DbDatabase: "drupal",
		},
	}
	if _, err := syncer.PrepareSyncer(); err != nil {
		t.Fatalf("PrepareSyncer() error: %v", err)
	}
	if err := syncer.SetTransferResource(containerDumpPath); err != nil {
		t.Fatalf("SetTransferResource() error: %v", err)
	}

	// GetLocalCommand uses the environment name to decide which config to use;
	// LOCAL_ENVIRONMENT_NAME triggers local overrides, so use a remote-style name here
	// so it reads from syncer.Config directly (same host, same creds).
	targetEnv := Environment{
		ProjectName:     "test-project",
		EnvironmentName: "test-env",
	}

	commands := syncer.GetLocalCommand(targetEnv)
	if len(commands) != 2 {
		t.Fatalf("expected GetLocalCommand to return 2 commands, got %d", len(commands))
	}

	for i, syncCmd := range commands {
		cmdStr, err := syncCmd.GetCommand()
		if err != nil {
			t.Fatalf("command[%d] GetCommand() error: %v", i, err)
		}
		execInContainer(ctx, t, container, cmdStr)
	}

	// Assert the Umami `node` table has at least one row.
	query := `mysql -h127.0.0.1 -udrupal -pdrupal -P3306 drupal -sNe "SELECT COUNT(*) FROM node;"`
	exitCode, output := execInContainerOutput(ctx, t, container, query)
	if exitCode != 0 {
		t.Fatalf("node count query failed (exit %d): %s", exitCode, output)
	}

	// The output from mysql -sN is prefixed with a spurious '+' byte from the
	// testcontainers multiplexed stream header; trim whitespace and control chars.
	count := strings.TrimSpace(output)
	count = strings.TrimLeft(count, "+\x00\x01\x02")
	count = strings.TrimSpace(count)
	t.Logf("node row count: %q", count)

	if count == "0" || count == "" {
		t.Fatalf("expected node table to contain rows after import, got count: %q", count)
	}
	t.Logf("SUCCESS: node table contains %s rows after import", count)
}
