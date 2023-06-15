package v1alpha1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"regexp"
)

var hasSpecial = regexp.MustCompile(`[!@#$%&*]+`).MatchString
var hasNumbers = regexp.MustCompile(`[0-9]+`).MatchString
var hasUppers = regexp.MustCompile(`[A-Z]+`).MatchString

var _ = Describe("Common", func() {

	Describe("GeneratePassword", func() {
		Describe("Special characters", func() {
			It("Should generate a password with special characters", func() {
				password := GeneratePassword(10, 1, 0, 0)
				Expect(password).ToNot(BeEmpty())
				Expect(hasSpecial(password)).To(BeTrue())
			})
			It("Should generate a password with special characters even with length of 1", func() {
				password := GeneratePassword(1, 1, 0, 0)
				Expect(password).ToNot(BeEmpty())
				Expect(hasSpecial(password)).To(BeTrue())
			})
		})
		Describe("Numbers", func() {
			It("Should generate a password with numeric characters", func() {
				password := GeneratePassword(10, 0, 1, 0)
				Expect(password).ToNot(BeEmpty())
				Expect(hasNumbers(password)).To(BeTrue())
			})
			It("Should generate a password with numeric characters even with length of 1", func() {
				password := GeneratePassword(1, 0, 1, 0)
				Expect(password).ToNot(BeEmpty())
				Expect(hasNumbers(password)).To(BeTrue())
			})
		})

		Describe("Uppercase", func() {
			It("Should generate a password with upper characters", func() {
				password := GeneratePassword(10, 0, 0, 1)
				Expect(password).ToNot(BeEmpty())
				Expect(hasUppers(password)).To(BeTrue())
			})
			It("Should generate a password with upper characters even with length of 1", func() {
				password := GeneratePassword(1, 0, 0, 1)
				Expect(password).ToNot(BeEmpty())
				Expect(hasUppers(password)).To(BeTrue())
			})
		})

		Describe("MinimumAllThree", func() {
			It("Should generate a password with upper, numeric and special characters", func() {
				password := GeneratePassword(10, 1, 1, 1)
				Expect(password).ToNot(BeEmpty())
				Expect(hasUppers(password)).To(BeTrue())
				Expect(hasNumbers(password)).To(BeTrue())
				Expect(hasSpecial(password)).To(BeTrue())
			})
			It("Should generate a password with upper, numeric and special characters even with length of 3", func() {
				password := GeneratePassword(3, 1, 1, 1)
				Expect(password).ToNot(BeEmpty())
				Expect(hasUppers(password)).To(BeTrue())
				Expect(hasNumbers(password)).To(BeTrue())
				Expect(hasSpecial(password)).To(BeTrue())
			})
		})
	})
})
