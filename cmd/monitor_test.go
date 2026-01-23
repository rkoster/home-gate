package cmd

import (
	"testing"

	fritzbox "home-gate/internal/fritzbox"
	"home-gate/internal/fritzbox/fritzboxfakes"
	"home-gate/internal/policy"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMonitor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Monitor Command Suite")
}

var _ = Describe("Monitor Command", func() {
	var fakeClient *fritzboxfakes.FakeClient
	var pm *policy.PolicyManager
	var landevices []fritzbox.Landevice
	var config fritzbox.MonitorConfig

	BeforeEach(func() {
		fakeClient = &fritzboxfakes.FakeClient{}
		var err error
		pm, err = policy.NewPolicyManager("MO-FR90")
		Expect(err).To(BeNil())

		landevices = []fritzbox.Landevice{
			{UID: "uid1", FriendlyName: "Device1", MAC: "00:11:22:33:44:55", UserUIDs: "user1", Blocked: "0"},
		}
		config = fritzbox.MonitorConfig{DisplayHomenetDevices: "uid1"}
	})

	Describe("RunMonitor", func() {
		It("should process devices for specific MAC", func() {
			// Setup fake client to return data
			fakeClient.GetMonitorDataReturns([]fritzbox.SubsetData{
				{DataSourceName: "rcv_001122334455", Measurements: []float64{1.0, 2.0}},
				{DataSourceName: "snd_001122334455", Measurements: []float64{0.5, 1.0}},
			}, nil)

			RunMonitor(fakeClient, pm, "00:11:22:33:44:55", "day", 0.0, landevices, config, map[string]string{}, false, GinkgoWriter)

			// Assert that GetMonitorData was called with correct params
			Expect(fakeClient.GetMonitorDataCallCount()).To(Equal(1))
			dataset, subset := fakeClient.GetMonitorDataArgsForCall(0)
			Expect(dataset).To(Equal("macaddrs"))
			Expect(subset).To(Equal("subset0002"))
		})

		It("should process configured devices when no MAC specified", func() {
			fakeClient.GetMonitorDataReturns([]fritzbox.SubsetData{
				{DataSourceName: "rcv_001122334455", Measurements: []float64{1.0}},
				{DataSourceName: "snd_001122334455", Measurements: []float64{0.5}},
			}, nil)

			RunMonitor(fakeClient, pm, "", "day", 0.0, landevices, config, map[string]string{}, false, GinkgoWriter)

			Expect(fakeClient.GetMonitorDataCallCount()).To(Equal(1))
			dataset, subset := fakeClient.GetMonitorDataArgsForCall(0)
			Expect(dataset).To(Equal("macaddrs"))
			Expect(subset).To(Equal("subset0002"))
		})

		It("should block device when exceeded and enforce is true", func() {
			// Set policy to low limit to exceed
			pm, _ := policy.NewPolicyManager("MO-FR20")
			fakeClient.GetMonitorDataReturns([]fritzbox.SubsetData{
				{DataSourceName: "rcv_001122334455", Measurements: []float64{1.0, 2.0}},
				{DataSourceName: "snd_001122334455", Measurements: []float64{0.5, 1.0}},
			}, nil)
			fakeClient.BlockDeviceReturns(nil)

			macToUserUID := map[string]string{"001122334455": "user1"}

			RunMonitor(fakeClient, pm, "00:11:22:33:44:55", "day", 0.0, landevices, config, macToUserUID, true, GinkgoWriter)

			Expect(fakeClient.BlockDeviceCallCount()).To(Equal(1))
			userUID, block := fakeClient.BlockDeviceArgsForCall(0)
			Expect(userUID).To(Equal("user1"))
			Expect(block).To(BeTrue())
		})

		It("should unblock device when within policy, blocked, and enforce is true", func() {
			// Device is blocked
			landevices[0].Blocked = "1"
			fakeClient.GetMonitorDataReturns([]fritzbox.SubsetData{
				{DataSourceName: "rcv_001122334455", Measurements: []float64{1.0, 2.0}},
				{DataSourceName: "snd_001122334455", Measurements: []float64{0.5, 1.0}},
			}, nil)
			fakeClient.BlockDeviceReturns(nil)

			macToUserUID := map[string]string{"001122334455": "user1"}

			RunMonitor(fakeClient, pm, "00:11:22:33:44:55", "day", 0.0, landevices, config, macToUserUID, true, GinkgoWriter)

			Expect(fakeClient.BlockDeviceCallCount()).To(Equal(1))
			userUID, block := fakeClient.BlockDeviceArgsForCall(0)
			Expect(userUID).To(Equal("user1"))
			Expect(block).To(BeFalse())
		})
	})
})
