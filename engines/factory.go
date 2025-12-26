/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package engines

import (
	"fmt"

	dbv1 "github.com/ivikasavnish/database-crd/api/v1"
	"github.com/ivikasavnish/database-crd/engines/postgres"
)

// DefaultEngineFactory is the default implementation of EngineFactory
type DefaultEngineFactory struct{}

// NewEngineFactory creates a new engine factory
func NewEngineFactory() EngineFactory {
	return &DefaultEngineFactory{}
}

// GetEngine returns an engine implementation for the specified database
func (f *DefaultEngineFactory) GetEngine(db *dbv1.Database) (Engine, error) {
	switch db.Spec.Engine {
	case dbv1.EnginePostgreSQL:
		return postgres.NewPostgresEngine(), nil
	case dbv1.EngineMongoDB:
		return nil, fmt.Errorf("MongoDB engine not yet implemented")
	case dbv1.EngineRedis:
		return nil, fmt.Errorf("Redis engine not yet implemented")
	case dbv1.EngineElasticsearch:
		return nil, fmt.Errorf("Elasticsearch engine not yet implemented")
	case dbv1.EngineSQLite:
		return nil, fmt.Errorf("SQLite engine not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported database engine: %s", db.Spec.Engine)
	}
}
