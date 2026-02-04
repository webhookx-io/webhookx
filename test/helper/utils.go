package helper

import (
	"os"
	"strings"
)

func SetEnvironments(envs map[string]string) {
	for name, value := range envs {
		if err := os.Setenv(name, value); err != nil {
			panic(err)
		}
	}
}

func ClearEnvironments(envs map[string]string) {
	for k := range envs {
		os.Unsetenv(k)
	}
}

func Env() map[string]string {
	envs := make(map[string]string)
	for _, env := range os.Environ() {
		if k, v, ok := strings.Cut(env, "="); ok {
			envs[k] = v
		}
	}
	return envs
}
