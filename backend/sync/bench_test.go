package sync

import (
	"fmt"
	"gosynctasks/backend"
	"gosynctasks/backend/sqlite"
	"path/filepath"
	"testing"
	"time"
)

// Benchmark tests for sync performance

// BenchmarkSyncPull benchmarks pulling tasks from remote
func BenchmarkSyncPull(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("tasks=%d", size), func(b *testing.B) {
			tmpDir := b.TempDir()
			dbPath := filepath.Join(tmpDir, "bench.db")

			localBackend, err := sqlite.NewSQLiteBackend(backend.BackendConfig{
				Type:    "sqlite",
				Enabled: true,
				DBPath:  dbPath,
			})
			if err != nil {
				b.Fatalf("Failed to create local backend: %v", err)
			}
			defer localBackend.Close()

			remoteBackend := backend.NewMockBackend()
			listID, _ := remoteBackend.CreateTaskList("Benchmark List", "", "")
			remoteBackend.Lists[0].CTags = "ctag-bench"

			// Pre-populate remote with tasks
			now := time.Now()
			for i := 0; i < size; i++ {
				remoteBackend.AddTask(listID, backend.Task{
					UID:      fmt.Sprintf("task-%d", i),
					Summary:  fmt.Sprintf("backend.Task %d", i),
					Status:   "NEEDS-ACTION",
					Priority: (i % 9) + 1,
					Created:  now,
					Modified: now,
				})
			}

			sm := NewSyncManager(localBackend, remoteBackend, ServerWins)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Clear local between runs
				localBackend.DeleteTaskList(listID)

				// Perform sync
				_, err := sm.Sync()
				if err != nil {
					b.Fatalf("Sync failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkSyncPush benchmarks pushing tasks to remote
func BenchmarkSyncPush(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("tasks=%d", size), func(b *testing.B) {
			tmpDir := b.TempDir()
			dbPath := filepath.Join(tmpDir, "bench.db")

			localBackend, err := sqlite.NewSQLiteBackend(backend.BackendConfig{
				Type:    "sqlite",
				Enabled: true,
				DBPath:  dbPath,
			})
			if err != nil {
				b.Fatalf("Failed to create local backend: %v", err)
			}
			defer localBackend.Close()

			remoteBackend := backend.NewMockBackend()
			listID, _ := localBackend.CreateTaskList("Benchmark List", "", "")
			remoteBackend.Lists = append(remoteBackend.Lists, backend.TaskList{
				ID:    listID,
				Name:  "Benchmark List",
				CTags: "ctag-bench",
			})
			remoteBackend.Tasks[listID] = []backend.Task{}

			sm := NewSyncManager(localBackend, remoteBackend, ServerWins)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				// Add tasks locally
				now := time.Now()
				for j := 0; j < size; j++ {
					localBackend.AddTask(listID, backend.Task{
						UID:      fmt.Sprintf("task-%d-%d", i, j),
						Summary:  fmt.Sprintf("backend.Task %d-%d", i, j),
						Status:   "NEEDS-ACTION",
						Priority: (j % 9) + 1,
						Created:  now,
						Modified: now,
					})
				}
				b.StartTimer()

				// Perform sync (push)
				_, err := sm.Sync()
				if err != nil {
					b.Fatalf("Sync failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkConflictResolution benchmarks different conflict resolution strategies
func BenchmarkConflictResolution(b *testing.B) {
	strategies := []ConflictResolutionStrategy{
		ServerWins,
		LocalWins,
		Merge,
		KeepBoth,
	}

	for _, strategy := range strategies {
		b.Run(string(strategy), func(b *testing.B) {
			tmpDir := b.TempDir()
			dbPath := filepath.Join(tmpDir, "bench.db")

			localBackend, err := sqlite.NewSQLiteBackend(backend.BackendConfig{
				Type:    "sqlite",
				Enabled: true,
				DBPath:  dbPath,
			})
			if err != nil {
				b.Fatalf("Failed to create local backend: %v", err)
			}
			defer localBackend.Close()

			remoteBackend := backend.NewMockBackend()

			// Create list
			listID, _ := localBackend.CreateTaskList("Conflict Bench", "", "")
			remoteBackend.Lists = append(remoteBackend.Lists, backend.TaskList{
				ID:    listID,
				Name:  "Conflict Bench",
				CTags: "ctag-initial",
			})
			remoteBackend.Tasks[listID] = []backend.Task{}

			sm := NewSyncManager(localBackend, remoteBackend, strategy)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				// Setup conflict
				now := time.Now()
				task := backend.Task{
					UID:      fmt.Sprintf("conflict-task-%d", i),
					Summary:  "Original",
					Status:   "NEEDS-ACTION",
					Priority: 5,
					Created:  now,
					Modified: now,
				}
				localBackend.AddTask(listID, task)

				// Modify locally
				task.Summary = "Local Modification"
				task.Priority = 1
				localBackend.UpdateTask(listID, task)

				// Modify remotely
				remoteTask := task
				remoteTask.Summary = "Remote Modification"
				remoteTask.Priority = 9
				remoteBackend.AddTask(listID, remoteTask)

				// Change CTag
				remoteBackend.Lists[0].CTags = fmt.Sprintf("ctag-%d", i)

				b.StartTimer()

				// Sync (resolve conflict)
				_, err := sm.Sync()
				if err != nil {
					b.Fatalf("Sync failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkDatabaseOperations benchmarks SQLite CRUD operations
func BenchmarkDatabaseOperations(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	sqliteBackend, err := sqlite.NewSQLiteBackend(backend.BackendConfig{
		Type:    "sqlite",
		Enabled: true,
		DBPath:  dbPath,
	})
	if err != nil {
		b.Fatalf("Failed to create sqliteBackend: %v", err)
	}
	defer sqliteBackend.Close()

	listID, _ := sqliteBackend.CreateTaskList("Benchmark List", "", "")

	b.Run("AddTask", func(b *testing.B) {
		now := time.Now()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sqliteBackend.AddTask(listID, backend.Task{
				UID:      fmt.Sprintf("task-%d", i),
				Summary:  fmt.Sprintf("backend.Task %d", i),
				Status:   "NEEDS-ACTION",
				Priority: (i % 9) + 1,
				Created:  now,
				Modified: now,
			})
		}
	})

	// Pre-populate for other benchmarks
	now := time.Now()
	taskUIDs := make([]string, 100)
	for i := 0; i < 100; i++ {
		uid := fmt.Sprintf("existing-task-%d", i)
		sqliteBackend.AddTask(listID, backend.Task{
			UID:      uid,
			Summary:  fmt.Sprintf("Existing backend.Task %d", i),
			Status:   "NEEDS-ACTION",
			Priority: 5,
			Created:  now,
			Modified: now,
		})
		taskUIDs[i] = uid
	}

	b.Run("GetTasks", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := sqliteBackend.GetTasks(listID, nil)
			if err != nil {
				b.Fatalf("GetTasks failed: %v", err)
			}
		}
	})

	b.Run("UpdateTask", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			uid := taskUIDs[i%len(taskUIDs)]
			sqliteBackend.UpdateTask(listID, backend.Task{
				UID:      uid,
				Summary:  fmt.Sprintf("Updated backend.Task %d", i),
				Status:   "COMPLETED",
				Priority: 1,
				Created:  now,
				Modified: now,
			})
		}
	})

	b.Run("FindTasksBySummary", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := sqliteBackend.FindTasksBySummary(listID, "Existing")
			if err != nil {
				b.Fatalf("FindTasksBySummary failed: %v", err)
			}
		}
	})

	b.Run("DeleteTask", func(b *testing.B) {
		// Pre-populate delete candidates
		deleteUIDs := make([]string, b.N)
		for i := 0; i < b.N; i++ {
			uid := fmt.Sprintf("delete-task-%d", i)
			sqliteBackend.AddTask(listID, backend.Task{
				UID:      uid,
				Summary:  "To Delete",
				Status:   "NEEDS-ACTION",
				Created:  now,
				Modified: now,
			})
			deleteUIDs[i] = uid
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sqliteBackend.DeleteTask(listID, deleteUIDs[i])
		}
	})
}

// BenchmarkSyncQueue benchmarks sync queue operations
func BenchmarkSyncQueue(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	sqliteBackend, err := sqlite.NewSQLiteBackend(backend.BackendConfig{
		Type:    "sqlite",
		Enabled: true,
		DBPath:  dbPath,
	})
	if err != nil {
		b.Fatalf("Failed to create sqliteBackend: %v", err)
	}
	defer sqliteBackend.Close()

	listID, _ := sqliteBackend.CreateTaskList("Queue Bench", "", "")

	b.Run("GetPendingSyncOperations", func(b *testing.B) {
		// Pre-populate queue
		for i := 0; i < 100; i++ {
			sqliteBackend.AddTask(listID, backend.Task{
				UID:      fmt.Sprintf("queue-task-%d", i),
				Summary:  "Queued backend.Task",
				Status:   "NEEDS-ACTION",
				Created:  time.Now(),
				Modified: time.Now(),
			})
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := sqliteBackend.GetPendingSyncOperations()
			if err != nil {
				b.Fatalf("GetPendingSyncOperations failed: %v", err)
			}
		}
	})

	b.Run("ClearSyncFlags", func(b *testing.B) {
		// Pre-populate
		taskUIDs := make([]string, b.N)
		for i := 0; i < b.N; i++ {
			uid := fmt.Sprintf("clear-task-%d", i)
			sqliteBackend.AddTask(listID, backend.Task{
				UID:      uid,
				Summary:  "backend.Task",
				Status:   "NEEDS-ACTION",
				Created:  time.Now(),
				Modified: time.Now(),
			})
			taskUIDs[i] = uid
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sqliteBackend.ClearSyncFlags(taskUIDs[i])
		}
	})
}

// BenchmarkHierarchicalTaskSorting benchmarks sorting tasks by hierarchy
func BenchmarkHierarchicalTaskSorting(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("tasks=%d", size), func(b *testing.B) {
			// Create hierarchical task structure
			now := time.Now()
			tasks := make([]backend.Task, size)

			// Create a tree: root tasks with children
			numRoots := size / 10
			if numRoots == 0 {
				numRoots = 1
			}

			for i := 0; i < size; i++ {
				task := backend.Task{
					UID:      fmt.Sprintf("task-%d", i),
					Summary:  fmt.Sprintf("backend.Task %d", i),
					Status:   "NEEDS-ACTION",
					Created:  now,
					Modified: now,
				}

				// Assign parent (skip first numRoots tasks as roots)
				if i >= numRoots {
					parentIdx := i % numRoots
					task.ParentUID = fmt.Sprintf("task-%d", parentIdx)
				}

				tasks[i] = task
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Create a copy to avoid modifying original
				tasksCopy := make([]backend.Task, len(tasks))
				copy(tasksCopy, tasks)

				// Sort by hierarchy
				_ = sortTasksByHierarchy(tasksCopy)
			}
		})
	}
}
