package policy_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"home-gate/internal/policy"
	"home-gate/internal/policy/policyfakes"
)

var _ = Describe("Policy", func() {

	Describe("NewPolicyManager", func() {
		It("should parse valid policy string", func() {
			pm, err := policy.NewPolicyManager("MO-TH90FR120SA-SU180")
			Expect(err).To(BeNil())
			Expect(pm).ToNot(BeNil())
		})

		It("should return error for invalid policy string", func() {
			pm, err := policy.NewPolicyManager("invalid")
			Expect(err).To(HaveOccurred())
			Expect(pm).To(BeNil())
		})
	})

	Describe("IsWithinPolicy", func() {
		var pm *policy.PolicyManager
		var fakeClock *policyfakes.FakeClock

		BeforeEach(func() {
			fakeClock = &policyfakes.FakeClock{}
			var err error
			pm, err = policy.NewPolicyManagerWithClock("MO-TH90FR120SA-SU180", fakeClock)
			Expect(err).To(BeNil())
		})

		Context("on Monday", func() {
			BeforeEach(func() {
				fakeClock.NowReturns(time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC)) // Monday
			})

			It("should be within policy if active minutes <= 90", func() {
				Expect(pm.IsWithinPolicy(80)).To(BeTrue())
			})

			It("should exceed policy if active minutes > 90", func() {
				Expect(pm.IsWithinPolicy(100)).To(BeFalse())
			})
		})

		Context("on Friday", func() {
			BeforeEach(func() {
				fakeClock.NowReturns(time.Date(2023, 1, 6, 12, 0, 0, 0, time.UTC)) // Friday
			})

			It("should be within policy if active minutes <= 120", func() {
				Expect(pm.IsWithinPolicy(110)).To(BeTrue())
			})

			It("should exceed policy if active minutes > 120", func() {
				Expect(pm.IsWithinPolicy(130)).To(BeFalse())
			})
		})

		Context("on Sunday", func() {
			BeforeEach(func() {
				fakeClock.NowReturns(time.Date(2023, 1, 8, 12, 0, 0, 0, time.UTC)) // Sunday
			})

			It("should be within policy if active minutes <= 180", func() {
				Expect(pm.IsWithinPolicy(170)).To(BeTrue())
			})

			It("should exceed policy if active minutes > 180", func() {
				Expect(pm.IsWithinPolicy(190)).To(BeFalse())
			})
		})
	})

})
