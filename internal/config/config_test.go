package config

import (
	"reflect"
	"testing"
)

func TestNewConfig(t *testing.T) {

	osArgs := []string{
		"-a", "localhost:5555",
		"-b", "http://localhost:5555",
		"-f", "/storageTest.csv",
		"-p", "./ssl",
		"-t", "0.0.0.0",
		"-s", "false",
		"-d", "user=ubuntu password=test101825 host=localhost port=5432 dbname=testdb"}

	envVars := map[string]string{
		"BASE_URL":          "http://localhost:5555",
		"SERVER_ADDRESS":    "localhost:5555",
		"FILE_STORAGE_PATH": "/storageTest.csv",
		"SSL_PATH":          "./ssl",
		"TRUSTED_SUBNET":    "0.0.0.0",
		"ENABLE_HTTPS":      "false",
		"DATABASE_DSN":      "user=ubuntu password=test101825 host=localhost port=5432 dbname=testdb",
	}

	filePath := "./testdata/1.json"

	tests := []struct {
		name string
		opts []configOption
		want Config
	}{
		{
			name: "flags only",
			opts: []configOption{WithEnvVars(map[string]string{}), withOsArgs(osArgs)},
			want: Config{
				baseURL:       "http://localhost:5555",
				srvAddr:       "localhost:5555",
				storagePath:   "/storageTest.csv",
				sslPath:       "./ssl",
				trustedSubnet: "0.0.0.0",
				dbDSN:         "user=ubuntu password=test101825 host=localhost port=5432 dbname=testdb",
			},
		},
		{
			name: "envs only",
			opts: []configOption{IgnoreOsArgs(), withOsArgs([]string{}), WithEnvVars(envVars)},
			want: Config{
				baseURL:       "http://localhost:5555",
				srvAddr:       "localhost:5555",
				storagePath:   "/storageTest.csv",
				sslPath:       "./ssl",
				trustedSubnet: "0.0.0.0",
				dbDSN:         "user=ubuntu password=test101825 host=localhost port=5432 dbname=testdb",
			},
		},
		{
			name: "file only",
			opts: []configOption{WithFile(filePath)},
			want: Config{
				baseURL:       "111",
				srvAddr:       "111",
				storagePath:   "111",
				dbDSN:         "111",
				sslPath:       "111",
				trustedSubnet: "111",
				useTLS:        true,
			},
		},
		{
			name: "flags over file",
			opts: []configOption{WithEnvVars(map[string]string{}), withOsArgs(osArgs), WithFile(filePath)},
			want: Config{
				baseURL:       "http://localhost:5555",
				srvAddr:       "localhost:5555",
				storagePath:   "/storageTest.csv",
				sslPath:       "./ssl",
				trustedSubnet: "0.0.0.0",
				dbDSN:         "user=ubuntu password=test101825 host=localhost port=5432 dbname=testdb",
				useTLS:        false,
			},
		},
		{
			name: "envs over file",
			opts: []configOption{IgnoreOsArgs(), WithFile(filePath), withOsArgs([]string{}), WithEnvVars(envVars)},
			want: Config{
				baseURL:       "http://localhost:5555",
				srvAddr:       "localhost:5555",
				storagePath:   "/storageTest.csv",
				sslPath:       "./ssl",
				trustedSubnet: "0.0.0.0",
				dbDSN:         "user=ubuntu password=test101825 host=localhost port=5432 dbname=testdb",
				useTLS:        false,
			},
		},
		{
			name: "flags over vars",
			opts: []configOption{withOsArgs(osArgs),
				WithEnvVars(map[string]string{
					"BASE_URL":          "111",
					"SERVER_ADDRESS":    "111",
					"FILE_STORAGE_PATH": "111",
					"SSL_PATH":          "111",
					"DATABASE_DSN":      "111",
					"ENABLE_HTTPS":      "111",
				})},
			want: Config{
				baseURL:       "http://localhost:5555",
				srvAddr:       "localhost:5555",
				storagePath:   "/storageTest.csv",
				sslPath:       "./ssl",
				trustedSubnet: "0.0.0.0",
				dbDSN:         "user=ubuntu password=test101825 host=localhost port=5432 dbname=testdb",
				useTLS:        false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.opts...); !reflect.DeepEqual(*got, tt.want) {
				t.Errorf("New().UseTLS() = %v, want %v", got.UseTLS(), tt.want.useTLS)
				t.Errorf("New().TrustedSubnet() = %v, want %v", got.TrustedSubnet(), tt.want.trustedSubnet)
				t.Errorf("New().SslPath() = %v, want %v", got.SslPath(), tt.want.sslPath)
				t.Errorf("New().BaseURL() = %v, want %v", got.BaseURL(), tt.want.baseURL)
				t.Errorf("New().DBDSN() = %v, want %v", got.DBDSN(), tt.want.dbDSN)
				t.Errorf("New().SrvAddr() = %v, want %v", got.SrvAddr(), tt.want.srvAddr)
				t.Errorf("New().StoragePath() = %v, want %v", got.StoragePath(), tt.want.storagePath)
			}
		})
	}
}
