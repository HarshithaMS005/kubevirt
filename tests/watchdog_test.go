/*
 * This file is part of the kubevirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017 Red Hat, Inc.
 *
 */

package tests_test

import (
	"runtime"
	"time"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/libvmops"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libvmifact"
)

var _ = Describe("[sig-compute]Watchdog", decorators.SigCompute, func() {

	Context("A VirtualMachineInstance with a watchdog device", func() {

		It("[test_id:4641]should be shut down when the watchdog expires", decorators.Conformance, func() {
			arch := runtime.GOARCH

			vmi := libvmops.RunVMIAndExpectLaunch(
				libvmifact.NewFedora(libvmi.WithWatchdog(v1.WatchdogActionPoweroff)), 360)

			By("Expecting the VirtualMachineInstance console")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			if arch == "s390x" {
				By("Loading the diag288_wdt watchdog module")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "sudo modprobe diag288_wdt\n"},
					&expect.BExp{R: console.PromptExpression},
				}, 250)).To(Succeed())

				By("Verifying if the module is loaded")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "lsmod | grep diag288_wdt\n"},
					&expect.BExp{R: "diag288_wdt"},
				}, 250)).To(Succeed())

			}

			By("Installing watchdog package")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "sudo dnf install -y watchdog\n"},
				&expect.BExp{R: console.PromptExpression},
			}, 250)).To(Succeed())

			By("Uncommenting watchdog-device line in /etc/watchdog.conf")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "sudo sed -i 's/^#watchdog-device/watchdog-device/' /etc/watchdog.conf\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "sudo sed -i 's/^watchdog-timeout.*/watchdog-timeout = 5/' /etc/watchdog.conf || echo 'watchdog-timeout = 5' | sudo tee -a /etc/watchdog.conf\n"},
				&expect.BExp{R: console.PromptExpression},
			}, 250)).To(Succeed())

			By("Starting watchdog service")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "sudo systemctl start watchdog\n"},
				&expect.BExp{R: console.PromptExpression},
			}, 250)).To(Succeed())

			//By("Verifying watchdog service is running")
			//Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
			//      &expect.BSnd{S: "systemctl status watchdog | grep 'Active:'\n"},
			//      &expect.BExp{R: "Active: active"},
			//}, 250)).To(Succeed())

			By("Checking watchdog device presence")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "ls /dev/watchdog\n"},
				&expect.BExp{R: "/dev/watchdog"},
			}, 250)).To(Succeed())

			//By("Checking watchdog service logs")
			//Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
			//      &expect.BSnd{S: "sudo journalctl -u watchdog --no-pager | tail -n 20\n"},
			//      &expect.BExp{R: console.PromptExpression}, // Ensures command execution completes
			//}, 250)).To(Succeed())

			By("Killing the watchdog device")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "sudo pkill -9 watchdog\n"},
				&expect.BExp{R: console.PromptExpression},
			}, 250)).To(Succeed())

			By("Waiting longer for watchdog to expire")
			time.Sleep(300 * time.Second) // Increased wait time

			By("Checking that the VirtualMachineInstance has Failed status")
			Eventually(matcher.ThisVMI(vmi)).WithTimeout(90 * time.Second).WithPolling(time.Second).
				Should(matcher.BeInPhase(v1.Failed))
		})

	})

})
