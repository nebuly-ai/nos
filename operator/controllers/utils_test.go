package controllers

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
