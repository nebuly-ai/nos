package utils

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"os"
)

var _ = Describe("BoolAddr", func() {
	When("Providing a boolean as input", func() {
		It("Should return false if the input is false", func() {
			var boolVar *bool
			Expect(BoolAddr(false)).To(BeAssignableToTypeOf(boolVar))
			Expect(BoolAddr(false)).To(HaveValue(BeFalse()))
		})
		It("Should return true if the input is true", func() {
			var boolVar *bool
			Expect(BoolAddr(true)).To(BeAssignableToTypeOf(boolVar))
			Expect(BoolAddr(true)).To(HaveValue(BeTrue()))
		})
	})
})

var _ = Describe("GetEnv", func() {
	When("Env variable is NOT defined", func() {
		It("Should return fallback value", func() {
			fallbackValue := "value"
			Expect(GetEnv("UNDEFINED", fallbackValue)).To(Equal(fallbackValue))
		})
	})

	When("Env variable is defined", func() {
		It("Should return the env variable value", func() {
			envVariableName := "ENV_VARIABLE"
			envVariableValue := "value"
			Expect(os.Setenv(envVariableName, envVariableValue)).To(Succeed())
			Expect(GetEnv(envVariableName, "fallback")).To(Equal(envVariableValue))
		})
	})
})
