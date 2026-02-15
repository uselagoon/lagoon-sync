package utils

import (
	"path/filepath"
	"testing"
)

func TestLoadDockerCompose(t *testing.T) {
	t.Run("valid docker-compose file", func(t *testing.T) {
		filePath := filepath.Join("test_data", "valid-compose.yml")
		services, err := LoadDockerCompose(filePath)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(services) != 2 {
			t.Errorf("Expected 2 services, got %d", len(services))
		}
		// Check for expected service names and types
		if web, ok := services["web"]; !ok || web.Type != "nginx-php" {
			t.Error("Expected 'web' service with type 'nginx-php'")
		}
		if db, ok := services["db"]; !ok || db.Type != "mariadb" {
			t.Error("Expected 'db' service with type 'mariadb'")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		nonExistentPath := filepath.Join("test_data", "non-existent.yml")
		services, err := LoadDockerCompose(nonExistentPath)
		if err == nil {
			t.Error("Expected error for non-existent file, got nil")
		}
		if len(services) > 0 {
			t.Error("Expected empty services for non-existent file")
		}
	})

	t.Run("invalid YAML", func(t *testing.T) {
		filePath := filepath.Join("test_data", "invalid-compose.yml")
		services, err := LoadDockerCompose(filePath)
		if err == nil {
			t.Error("Expected error for invalid YAML, got nil")
		}
		if len(services) > 0 {
			t.Error("Expected empty services for invalid YAML")
		}
	})

	t.Run("empty services", func(t *testing.T) {
		filePath := filepath.Join("test_data", "empty-compose.yml")
		services, err := LoadDockerCompose(filePath)
		if err != nil {
			t.Errorf("Expected no error for empty services, got: %v", err)
		}
		if len(services) > 0 {
			t.Error("Expected empty services map")
		}
	})

	t.Run("lagoon.persistent volume", func(t *testing.T) {
		filePath := filepath.Join("test_data", "volumes-compose.yml")
		services, err := LoadDockerCompose(filePath)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		app, ok := services["app"]
		if !ok {
			t.Fatal("Expected 'app' service")
		}
		if vol, exists := app.Volumes["/app/files"]; !exists || vol != "/app/files" {
			t.Error("Expected lagoon.persistent volume '/app/files' to be set")
		}
	})

	t.Run("lagoon.volumes.*.path pattern", func(t *testing.T) {
		filePath := filepath.Join("test_data", "volumes-compose.yml")
		services, err := LoadDockerCompose(filePath)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		app, ok := services["app"]
		if !ok {
			t.Fatal("Expected 'app' service")
		}
		if vol, exists := app.Volumes["config"]; !exists || vol != "/app/config" {
			t.Error("Expected lagoon.volumes.config.path to set volume 'config' to '/app/config'")
		}
		if vol, exists := app.Volumes["data"]; !exists || vol != "/app/data" {
			t.Error("Expected lagoon.volumes.data.path to set volume 'data' to '/app/data'")
		}
	})
}
