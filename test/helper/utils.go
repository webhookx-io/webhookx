package helper

import "os"

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
