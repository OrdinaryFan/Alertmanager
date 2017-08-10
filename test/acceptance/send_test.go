// Copyright 2015 Prometheus Team
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	. "github.com/prometheus/alertmanager/test"
)

// This file contains acceptance tests around the basic sending logic
// for notifications, which includes batching and ensuring that each
// notification is eventually sent at least once and ideally exactly
// once.

func TestMergeAlerts(t *testing.T) {
	t.Parallel()

	conf := `
route:
  receiver: "default"
  group_by: []
  group_wait:      1s
  group_interval:  1s
  repeat_interval: 0s

receivers:
- name: "default"
  webhook_configs:
  - url: 'http://%s'
    send_resolved: true
`

	at := NewAcceptanceTest(t, &AcceptanceOpts{
		Tolerance: 150 * time.Millisecond,
	})

	co := at.Collector("webhook")
	wh := NewWebhook(co)

	am := at.Alertmanager(fmt.Sprintf(conf, wh.Address()))

	// Refresh an alert several times. The starting time must remain at the earliest
	// point in time.
	am.Push(At(1), Alert("alertname", "test").Active(1.1))
	// Another Prometheus server might be sending later but with an earlier start time.
	am.Push(At(1.2), Alert("alertname", "test").Active(1))

	co.Want(Between(2, 2.5), Alert("alertname", "test").Active(1))

	am.Push(At(2.1), Alert("alertname", "test").Annotate("ann", "v1").Active(2))

	co.Want(Between(3, 3.5), Alert("alertname", "test").Annotate("ann", "v1").Active(1))

	// Annotations are always overwritten by the alert that arrived most recently.
	am.Push(At(3.6), Alert("alertname", "test").Annotate("ann", "v2").Active(1.5))

	co.Want(Between(4, 4.5), Alert("alertname", "test").Annotate("ann", "v2").Active(1))

	// If an alert is marked resolved twice, the latest point in time must be
	// set as the eventual resolve time.
	am.Push(At(4.6), Alert("alertname", "test").Annotate("ann", "v2").Active(3, 4.5))
	am.Push(At(4.8), Alert("alertname", "test").Annotate("ann", "v3").Active(2.9, 4.8))
	am.Push(At(4.8), Alert("alertname", "test").Annotate("ann", "v3").Active(2.9, 4.1))

	co.Want(Between(5, 5.5), Alert("alertname", "test").Annotate("ann", "v3").Active(1, 4.8))

	// Reactivate an alert after a previous occurrence has been resolved.
	// No overlap, no merge must occur.
	am.Push(At(5.3), Alert("alertname", "test"))

	co.Want(Between(6, 6.5), Alert("alertname", "test").Active(5.3))

	// Test against a bug which ocurrec after a restart. The previous occurrence of
	// the alert was sent rather than the most recent one.
	//
	// XXX(fabxc) disabled as notification info won't be persisted. Thus, with a mesh
	// notifier we lose the state in this single-node setup.
	//at.Do(At(6.7), func() {
	//	am.Terminate()
	//	am.Start()
	//})

	// On restart the alert is flushed right away as the group_wait has already passed.
	// However, it must be caught in the deduplication stage.
	// The next attempt will be 1s later and won't be filtered in deduping.
	//co.Want(Between(7.7, 8), Alert("alertname", "test").Active(5.3))

	at.Run()
}

func TestRepeat(t *testing.T) {
	t.Parallel()

	conf := `
route:
  receiver: "default"
  group_by: []
  group_wait:      1s
  group_interval:  1s
  repeat_interval: 0s 

receivers:
- name: "default"
  webhook_configs:
  - url: 'http://%s'
`

	// Create a new acceptance test that instantiates new Alertmanagers
	// with the given configuration and verifies times with the given
	// tollerance.
	at := NewAcceptanceTest(t, &AcceptanceOpts{
		Tolerance: 150 * time.Millisecond,
	})

	// Create a collector to which alerts can be written and verified
	// against a set of expected alert notifications.
	co := at.Collector("webhook")
	// Run something that satisfies the webhook interface to which the
	// Alertmanager pushes as defined by its configuration.
	wh := NewWebhook(co)

	// Create a new Alertmanager process listening to a random port
	am := at.Alertmanager(fmt.Sprintf(conf, wh.Address()))

	// Declare pushes to be made to the Alertmanager at the given time.
	// Times are provided in fractions of seconds.
	am.Push(At(1), Alert("alertname", "test").Active(1))

	// XXX(fabxc): disabled as long as alerts are not persisted.
	// at.Do(At(1.2), func() {
	//	am.Terminate()
	//	am.Start()
	// })
	am.Push(At(3.5), Alert("alertname", "test").Active(1, 3))

	// Declare which alerts are expected to arrive at the collector within
	// the defined time intervals.
	co.Want(Between(2, 2.5), Alert("alertname", "test").Active(1))
	co.Want(Between(3, 3.5), Alert("alertname", "test").Active(1))
	co.Want(Between(4, 4.5), Alert("alertname", "test").Active(1, 3))

	// Start the flow as defined above and run the checks afterwards.
	at.Run()
}

func TestRetry(t *testing.T) {
	t.Parallel()

	// We create a notification config that fans out into two different
	// webhooks.
	// The succeeding one must still only receive the first successful
	// notifications. Sending to the succeeding one must eventually succeed.
	conf := `
route:
  receiver: "default"
  group_by: []
  group_wait:      1s
  group_interval:  1s
  repeat_interval: 3s

receivers:
- name: "default"
  webhook_configs:
  - url: 'http://%s'
  - url: 'http://%s'
`

	at := NewAcceptanceTest(t, &AcceptanceOpts{
		Tolerance: 150 * time.Millisecond,
	})

	co1 := at.Collector("webhook")
	wh1 := NewWebhook(co1)

	co2 := at.Collector("webhook_failing")
	wh2 := NewWebhook(co2)

	wh2.Func = func(ts float64) bool {
		// Fail the first two interval periods but eventually
		// succeed in the third interval after a few failed attempts.
		return ts < 4.5
	}

	am := at.Alertmanager(fmt.Sprintf(conf, wh1.Address(), wh2.Address()))

	am.Push(At(1), Alert("alertname", "test1"))

	co1.Want(Between(2, 2.5), Alert("alertname", "test1").Active(1))
	co1.Want(Between(5, 5.5), Alert("alertname", "test1").Active(1))

	co2.Want(Between(4.5, 5), Alert("alertname", "test1").Active(1))
}

func TestBatching(t *testing.T) {
	t.Parallel()

	conf := `
route:
  receiver: "default"
  group_by: []
  group_wait:      1s
  group_interval:  1s
  repeat_interval: 5s

receivers:
- name: "default"
  webhook_configs:
  - url: 'http://%s'
`

	at := NewAcceptanceTest(t, &AcceptanceOpts{
		Tolerance: 150 * time.Millisecond,
	})

	co := at.Collector("webhook")
	wh := NewWebhook(co)

	am := at.Alertmanager(fmt.Sprintf(conf, wh.Address()))

	am.Push(At(1.1), Alert("alertname", "test1").Active(1))
	am.Push(At(1.7), Alert("alertname", "test5").Active(1))

	co.Want(Between(2.0, 2.5),
		Alert("alertname", "test1").Active(1),
		Alert("alertname", "test5").Active(1),
	)

	am.Push(At(3.3),
		Alert("alertname", "test2").Active(1.5),
		Alert("alertname", "test3").Active(1.5),
		Alert("alertname", "test4").Active(1.6),
	)

	co.Want(Between(4.1, 4.5),
		Alert("alertname", "test1").Active(1),
		Alert("alertname", "test5").Active(1),
		Alert("alertname", "test2").Active(1.5),
		Alert("alertname", "test3").Active(1.5),
		Alert("alertname", "test4").Active(1.6),
	)

	// While no changes happen expect no additional notifications
	// until the 5s repeat interval has ended.

	co.Want(Between(9.1, 9.5),
		Alert("alertname", "test1").Active(1),
		Alert("alertname", "test5").Active(1),
		Alert("alertname", "test2").Active(1.5),
		Alert("alertname", "test3").Active(1.5),
		Alert("alertname", "test4").Active(1.6),
	)

	at.Run()
}

func TestResolved(t *testing.T) {
	t.Parallel()

	var wg sync.WaitGroup
	wg.Add(10)

	for i := 0; i < 10; i++ {
		go func() {
			conf := `
global:
  resolve_timeout: 10s

route:
  receiver: "default"
  group_by: [alertname]
  group_wait: 1s 
  group_interval: 5s

receivers:
- name: "default"
  webhook_configs:
  - url: 'http://%s'
`

			at := NewAcceptanceTest(t, &AcceptanceOpts{
				Tolerance: 150 * time.Millisecond,
			})

			co := at.Collector("webhook")
			wh := NewWebhook(co)

			am := at.Alertmanager(fmt.Sprintf(conf, wh.Address()))

			am.Push(At(1),
				Alert("alertname", "test", "lbl", "v1"),
				Alert("alertname", "test", "lbl", "v2"),
				Alert("alertname", "test", "lbl", "v3"),
			)

			co.Want(Between(2, 2.5),
				Alert("alertname", "test", "lbl", "v1").Active(1),
				Alert("alertname", "test", "lbl", "v2").Active(1),
				Alert("alertname", "test", "lbl", "v3").Active(1),
			)
			co.Want(Between(12, 13),
				Alert("alertname", "test", "lbl", "v1").Active(1, 11),
				Alert("alertname", "test", "lbl", "v2").Active(1, 11),
				Alert("alertname", "test", "lbl", "v3").Active(1, 11),
			)

			at.Run()
			wg.Done()
		}()
	}

	wg.Wait()
}

func TestResolvedFilter(t *testing.T) {
	t.Parallel()

	// This integration test ensures that even though resolved alerts may not be
	// notified about, they must be set as notified. Resolved alerts, even when
	// filtered, have to end up in the SetNotifiesStage, otherwise when an alert
	// fires again it is ambiguous whether it was resolved in between or not.

	conf := `
global:
  resolve_timeout: 10s

route:
  receiver: "default"
  group_by: [alertname]
  group_wait: 1s
  group_interval: 5s

receivers:
- name: "default"
  webhook_configs:
  - url: 'http://%s'
    send_resolved: true
  - url: 'http://%s'
    send_resolved: false
`

	at := NewAcceptanceTest(t, &AcceptanceOpts{
		Tolerance: 150 * time.Millisecond,
	})

	co1 := at.Collector("webhook1")
	wh1 := NewWebhook(co1)

	co2 := at.Collector("webhook2")
	wh2 := NewWebhook(co2)

	am := at.Alertmanager(fmt.Sprintf(conf, wh1.Address(), wh2.Address()))

	am.Push(At(1),
		Alert("alertname", "test", "lbl", "v1"),
		Alert("alertname", "test", "lbl", "v2"),
	)
	am.Push(At(3),
		Alert("alertname", "test", "lbl", "v1").Active(1, 4),
		Alert("alertname", "test", "lbl", "v3"),
	)
	am.Push(At(8),
		Alert("alertname", "test", "lbl", "v3").Active(3),
	)

	co1.Want(Between(2, 2.5),
		Alert("alertname", "test", "lbl", "v1").Active(1),
		Alert("alertname", "test", "lbl", "v2").Active(1),
	)
	co1.Want(Between(7, 7.5),
		Alert("alertname", "test", "lbl", "v1").Active(1, 4),
		Alert("alertname", "test", "lbl", "v2").Active(1),
		Alert("alertname", "test", "lbl", "v3").Active(3),
	)
	co1.Want(Between(12, 12.5),
		Alert("alertname", "test", "lbl", "v2").Active(1, 11),
		Alert("alertname", "test", "lbl", "v3").Active(3),
	)

	co2.Want(Between(2, 2.5),
		Alert("alertname", "test", "lbl", "v1").Active(1),
		Alert("alertname", "test", "lbl", "v2").Active(1),
	)
	co2.Want(Between(7, 7.5),
		Alert("alertname", "test", "lbl", "v2").Active(1),
		Alert("alertname", "test", "lbl", "v3").Active(3),
	)

	at.Run()
}

func TestReload(t *testing.T) {
	t.Parallel()

	// We create a notification config that fans out into two different
	// webhooks.
	// The succeeding one must still only receive the first successful
	// notifications. Sending to the succeeding one must eventually succeed.
	conf := `
route:
  receiver: "default"
  group_by: []
  group_wait:      1s
  group_interval:  6s
  repeat_interval: 10m

receivers:
- name: "default"
  webhook_configs:
  - url: 'http://%s'
`

	at := NewAcceptanceTest(t, &AcceptanceOpts{
		Tolerance: 150 * time.Millisecond,
	})

	co := at.Collector("webhook")
	wh := NewWebhook(co)

	am := at.Alertmanager(fmt.Sprintf(conf, wh.Address()))

	am.Push(At(1), Alert("alertname", "test1"))
	at.Do(At(3), am.Reload)
	am.Push(At(4), Alert("alertname", "test2"))

	co.Want(Between(2, 2.5), Alert("alertname", "test1").Active(1))
	// Timers are reset on reload regardless, so we count the 6 second group
	// interval from 3 onwards.
	co.Want(Between(9, 9.5),
		Alert("alertname", "test1").Active(1),
		Alert("alertname", "test2").Active(4),
	)

	at.Run()
}
