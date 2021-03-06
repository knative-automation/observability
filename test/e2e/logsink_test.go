// +build e2e

/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"fmt"
	"testing"

	"github.com/knative/observability/pkg/apis/sink/v1alpha1"
	observabilityv1alpha1 "github.com/knative/observability/pkg/client/clientset/versioned/typed/sink/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSyslogLogSink(t *testing.T) {
	var prefix = randomTestPrefix("syslog-log-sink-")

	clients := initialize(t)
	defer teardownNamespaces(t, clients)

	cleanup := createSyslogLogSink(t, prefix, clients.sinkClient, observabilityTestNamespace)
	defer cleanup()
	createSyslogReceiver(t, prefix, clients.kubeClient, observabilityTestNamespace)
	waitForFluentBitToBeReady(t, prefix, clients.kubeClient)
	emitLogs(t, prefix, clients.kubeClient, observabilityTestNamespace)
	emitLogs(t, prefix, clients.kubeClient, crosstalkTestNamespace)
	assertOnCrosstalk(t, prefix, clients, observabilityTestNamespace, func(m ReceiverMetrics) error {
		if m.Cluster != 10 {
			return fmt.Errorf("cluster count != 10")
		}
		messagesObservability, ok := m.Namespaced[observabilityTestNamespace]
		if !ok || messagesObservability != 10 {
			return fmt.Errorf("test namespace count != 10")
		}
		_, ok = m.Namespaced[crosstalkTestNamespace]
		if ok {
			return fmt.Errorf("crosstalk namespace messages came through")
		}
		return nil
	},
	)
}

func TestEventsLogSink(t *testing.T) {
	var prefix = randomTestPrefix("event-log-sink-")

	clients := initialize(t)
	defer teardownNamespaces(t, clients)

	t.Logf("Test Prefix: %s", prefix)
	cleanup := createSyslogLogSink(t, prefix, clients.sinkClient, observabilityTestNamespace)
	defer cleanup()
	createSyslogReceiver(t, prefix, clients.kubeClient, observabilityTestNamespace)
	waitForFluentBitToBeReady(t, prefix, clients.kubeClient)
	numEvents := 100
	emitEvents(t, "clearing-event-controller", clients.kubeClient, observabilityTestNamespace, numEvents)
	emitEvents(t, prefix, clients.kubeClient, observabilityTestNamespace, numEvents)
	emitEvents(t, prefix, clients.kubeClient, crosstalkTestNamespace, numEvents)
	assertOnCrosstalk(t, prefix, clients, observabilityTestNamespace, func(m ReceiverMetrics) error {
		if m.Cluster != numEvents {
			return fmt.Errorf("cluster numEvents != %d", numEvents)
		}
		messagesObservability, ok := m.Namespaced[observabilityTestNamespace]
		if !ok || messagesObservability != numEvents {
			return fmt.Errorf("test namespace numEvents != %d", numEvents)
		}
		_, ok = m.Namespaced[crosstalkTestNamespace]
		if ok {
			return fmt.Errorf("crosstalk namespace messages came through")
		}
		return nil
	},
	)
}

func TestWebhookLogSink(t *testing.T) {
	var prefix = randomTestPrefix("webhook-log-sink-")

	clients := initialize(t)
	defer teardownNamespaces(t, clients)

	t.Logf("Test Prefix: %s", prefix)
	cleanup := createWebhookLogSink(t, prefix, clients.sinkClient, observabilityTestNamespace)
	defer cleanup()
	createSyslogReceiver(t, prefix, clients.kubeClient, observabilityTestNamespace)
	waitForFluentBitToBeReady(t, prefix, clients.kubeClient)
	emitLogs(t, prefix, clients.kubeClient, observabilityTestNamespace)
	emitLogs(t, prefix, clients.kubeClient, crosstalkTestNamespace)
	assertOnCrosstalk(t, prefix, clients, observabilityTestNamespace, func(m ReceiverMetrics) error {
		messagesObservability, ok := m.WebhookNamespaced[observabilityTestNamespace]
		if !ok || messagesObservability < 10 {
			return fmt.Errorf("test namespace messages < 10")
		}
		_, ok = m.WebhookNamespaced[crosstalkTestNamespace]
		if ok {
			return fmt.Errorf("crosstalk namespace messages came through")
		}
		return nil
	})
}

func TestCrosstalkLogSink(t *testing.T) {
	var prefix = randomTestPrefix("test-crosstalk-logsink")

	clients := initialize(t)
	defer teardownNamespaces(t, clients)

	cleanup1 := createSyslogLogSink(t, prefix, clients.sinkClient, observabilityTestNamespace)
	defer cleanup1()
	cleanup2 := createSyslogLogSink(t, prefix, clients.sinkClient, crosstalkTestNamespace)
	defer cleanup2()
	createSyslogReceiver(t, prefix, clients.kubeClient, observabilityTestNamespace)
	createSyslogReceiver(t, prefix, clients.kubeClient, crosstalkTestNamespace)
	waitForFluentBitToBeReady(t, prefix, clients.kubeClient)
	emitLogs(t, prefix, clients.kubeClient, observabilityTestNamespace)
	emitLogs(t, prefix, clients.kubeClient, crosstalkTestNamespace)
	assertOnCrosstalk(t, prefix, clients, observabilityTestNamespace, func(m ReceiverMetrics) error {
		if m.Cluster != 10 {
			return fmt.Errorf("cluster count != 10")
		}
		messagesObservability, ok := m.Namespaced[observabilityTestNamespace]
		if !ok || messagesObservability != 10 {
			return fmt.Errorf("test namespace count != 10")
		}
		_, ok = m.Namespaced[crosstalkTestNamespace]
		if ok {
			return fmt.Errorf("crosstalk namespace messages came through")
		}
		return nil
	},
	)
	assertOnCrosstalk(t, prefix, clients, crosstalkTestNamespace, func(m ReceiverMetrics) error {
		if m.Cluster != 10 {
			return fmt.Errorf("cluster count != 10")
		}
		messagesObservability, ok := m.Namespaced[crosstalkTestNamespace]
		if !ok || messagesObservability != 10 {
			return fmt.Errorf("crosstalk namespace count != 10")
		}
		_, ok = m.Namespaced[observabilityTestNamespace]
		if ok {
			return fmt.Errorf("observability namespace messages came through")
		}
		return nil
	},
	)
}

func createSyslogLogSink(
	t *testing.T,
	prefix string,
	sc observabilityv1alpha1.ObservabilityV1alpha1Interface,
	namespace string,
) func() error {
	t.Log("Creating the syslog LogSink")
	_, err := sc.LogSinks(namespace).Create(&v1alpha1.LogSink{
		ObjectMeta: metav1.ObjectMeta{
			Name:      prefix + "test",
			Namespace: namespace,
		},
		Spec: v1alpha1.SinkSpec{
			Type: "syslog",
			SyslogSpec: v1alpha1.SyslogSpec{
				Host:      prefix + syslogReceiverSuffix + "." + namespace,
				Port:      24903,
				EnableTLS: true,
			},
			InsecureSkipVerify: true,
		},
	})
	assertErr(t, "Error creating syslog LogSink: %v", err)

	return func() error {
		return sc.LogSinks(namespace).Delete(prefix+"test", &metav1.DeleteOptions{})
	}
}

func createWebhookLogSink(
	t *testing.T,
	prefix string,
	sc observabilityv1alpha1.ObservabilityV1alpha1Interface,
	namespace string,
) func() error {
	t.Log("Creating the webhook LogSink")
	_, err := sc.LogSinks(namespace).Create(&v1alpha1.LogSink{
		ObjectMeta: metav1.ObjectMeta{
			Name:      prefix + "test",
			Namespace: namespace,
		},
		Spec: v1alpha1.SinkSpec{
			Type: "webhook",
			WebhookSpec: v1alpha1.WebhookSpec{
				URL: "https://" + prefix + syslogReceiverSuffix + "." + namespace + ":7070/webhook",
			},
			InsecureSkipVerify: true,
		},
	})
	assertErr(t, "Error creating webhook LogSink: %v", err)

	return func() error {
		return sc.LogSinks(namespace).Delete(prefix+"test", &metav1.DeleteOptions{})
	}
}
