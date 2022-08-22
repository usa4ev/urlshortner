package config

import (
	"reflect"
	"testing"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name string
		opts []configOption
		want Config
	}{
		{
			name: "flags only",
			opts: []configOption{WithEnvVars(map[string]string{}), withOsArgs([]string{"-a", "localhost:5555", "-b", "http://localhost:5555", "-f", "/storageTest.csv", "-d", "user=ubuntu password=test101825 host=localhost port=5432 dbname=testdb"})},
			want: Config{
				baseURL:     "http://localhost:5555",
				srvAddr:     "localhost:5555",
				storagePath: "/storageTest.csv",
				dbDSN:       "user=ubuntu password=test101825 host=localhost port=5432 dbname=testdb",
			},
		},
		{
			name: "envs only",
			opts: []configOption{IgnoreOsArgs(), withOsArgs([]string{}), WithEnvVars(map[string]string{
				"BASE_URL":          "http://localhost:5555",
				"SERVER_ADDRESS":    "localhost:5555",
				"FILE_STORAGE_PATH": "/storageTest.csv",
				"DATABASE_DSN":      "user=ubuntu password=test101825 host=localhost port=5432 dbname=testdb",
			})},
			want: Config{
				baseURL:     "http://localhost:5555",
				srvAddr:     "localhost:5555",
				storagePath: "/storageTest.csv",
				dbDSN:       "user=ubuntu password=test101825 host=localhost port=5432 dbname=testdb",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.opts...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}
