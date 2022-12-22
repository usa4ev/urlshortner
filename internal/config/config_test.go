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
			opts: []configOption{WithEnvVars(map[string]string{}), withOsArgs([]string{"-a", "localhost:5555", "-b", "http://localhost:5555", "-f", "/storageTest.csv", "-p", "./ssl", "-d", "user=ubuntu password=test101825 host=localhost port=5432 dbname=testdb"})},
			want: Config{
				baseURL:     "http://localhost:5555",
				srvAddr:     "localhost:5555",
				storagePath: "/storageTest.csv",
				sslPath:     "./ssl",
				dbDSN:       "user=ubuntu password=test101825 host=localhost port=5432 dbname=testdb",
			},
		},
		{
			name: "envs only",
			opts: []configOption{IgnoreOsArgs(), withOsArgs([]string{}), WithEnvVars(map[string]string{
				"BASE_URL":          "http://localhost:5555",
				"SERVER_ADDRESS":    "localhost:5555",
				"FILE_STORAGE_PATH": "/storageTest.csv",
				"SSL_PATH":          "./ssl",
				"DATABASE_DSN":      "user=ubuntu password=test101825 host=localhost port=5432 dbname=testdb",
			})},
			want: Config{
				baseURL:     "http://localhost:5555",
				srvAddr:     "localhost:5555",
				storagePath: "/storageTest.csv",
				sslPath:     "./ssl",
				dbDSN:       "user=ubuntu password=test101825 host=localhost port=5432 dbname=testdb",
			},
		},
		{
			name: "file only",
			opts: []configOption{WithFile("./testdata/1.json")},
			want: Config{
				baseURL:     "111",
				srvAddr:     "111",
				storagePath: "111",
				dbDSN:       "111",
				sslPath:     "111",
				useTLS:      true,
			},
		},
		{
			name: "flags over file",
			opts: []configOption{WithEnvVars(map[string]string{}), withOsArgs([]string{"-a", "localhost:5555", "-b", "http://localhost:5555", "-f", "/storageTest.csv", "-p", "./ssl", "-d", "user=ubuntu password=test101825 host=localhost port=5432 dbname=testdb", "-s", "false"}), WithFile("./testdata/1.json")},
			want: Config{
				baseURL:     "http://localhost:5555",
				srvAddr:     "localhost:5555",
				storagePath: "/storageTest.csv",
				sslPath:     "./ssl",
				dbDSN:       "user=ubuntu password=test101825 host=localhost port=5432 dbname=testdb",

				useTLS: false,
			},
		},
		{
			name: "envs over file",
			opts: []configOption{IgnoreOsArgs(), WithFile("./testdata/1.json"), withOsArgs([]string{}), WithEnvVars(map[string]string{
				"BASE_URL":          "http://localhost:5555",
				"SERVER_ADDRESS":    "localhost:5555",
				"FILE_STORAGE_PATH": "/storageTest.csv",
				"SSL_PATH":          "./ssl",
				"DATABASE_DSN":      "user=ubuntu password=test101825 host=localhost port=5432 dbname=testdb",
				"ENABLE_HTTPS":      "false",
			})},
			want: Config{
				baseURL:     "http://localhost:5555",
				srvAddr:     "localhost:5555",
				storagePath: "/storageTest.csv",
				sslPath:     "./ssl",
				dbDSN:       "user=ubuntu password=test101825 host=localhost port=5432 dbname=testdb",
				useTLS:      false,
			},
		},
		{
			name: "flags over vars",
			opts: []configOption{withOsArgs([]string{"-a", "localhost:5555", "-b", "http://localhost:5555", "-f", "/storageTest.csv", "-p", "./ssl", "-d", "user=ubuntu password=test101825 host=localhost port=5432 dbname=testdb"}),
				WithEnvVars(map[string]string{
					"BASE_URL":          "111",
					"SERVER_ADDRESS":    "111",
					"FILE_STORAGE_PATH": "111",
					"SSL_PATH":          "111",
					"DATABASE_DSN":      "111",
					"ENABLE_HTTPS":      "111",
				})},
			want: Config{
				baseURL:     "http://localhost:5555",
				srvAddr:     "localhost:5555",
				storagePath: "/storageTest.csv",
				sslPath:     "./ssl",
				dbDSN:       "user=ubuntu password=test101825 host=localhost port=5432 dbname=testdb",
				useTLS:      false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.opts...); !reflect.DeepEqual(*got, tt.want) {
				t.Errorf("New() = %v, want %v", *got, tt.want)
			}
		})
	}
}
