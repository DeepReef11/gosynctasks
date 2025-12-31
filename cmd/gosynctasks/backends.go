package main

// This file imports all backend implementations to trigger their init() functions,
// which register them with the global backend registry.
//
// The blank imports ensure that all backends are registered at program startup.

import (
	_ "gosynctasks/backend/file"      // File backend
	_ "gosynctasks/backend/git"       // Git backend
	_ "gosynctasks/backend/nextcloud" // Nextcloud backend
	_ "gosynctasks/backend/sqlite"    // SQLite backend
)
