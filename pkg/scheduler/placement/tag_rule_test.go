/*
 Licensed to the Apache Software Foundation (ASF) under one
 or more contributor license agreements.  See the NOTICE file
 distributed with this work for additional information
 regarding copyright ownership.  The ASF licenses this file
 to you under the Apache License, Version 2.0 (the
 "License"); you may not use this file except in compliance
 with the License.  You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package placement

import (
	"testing"

	"gotest.tools/assert"

	"github.com/apache/incubator-yunikorn-core/pkg/cache"
	"github.com/apache/incubator-yunikorn-core/pkg/common/configs"
	"github.com/apache/incubator-yunikorn-core/pkg/common/security"
)

func TestTagRule(t *testing.T) {
	conf := configs.PlacementRule{
		Name: "tag",
	}
	tr, err := newRule(conf)
	if err == nil || tr != nil {
		t.Errorf("tag rule create did not fail without tag name, err 'nil' , rule: %v, ", tr)
	}
	conf = configs.PlacementRule{
		Name:  "tag",
		Value: "label1",
	}
	tr, err = newRule(conf)
	if err != nil || tr == nil {
		t.Errorf("tag rule create failed with tag name, err %v", err)
	}
	// trying to create using a parent with a fully qualified child
	conf = configs.PlacementRule{
		Name:  "tag",
		Value: "label1",
		Parent: &configs.PlacementRule{
			Name:  "tag",
			Value: "label2",
		},
	}
	tr, err = newRule(conf)
	if err != nil || tr == nil {
		t.Errorf("tag rule create failed with tag as parent rule, err %v", err)
	}
}

func TestTagRulePlace(t *testing.T) {
	// Create the structure for the test
	data := `
partitions:
  - name: default
    queues:
      - name: testqueue
      - name: testparent
        queues:
          - name: testchild
`
	partInfo, err := CreatePartitionInfo([]byte(data))
	assert.NilError(t, err, "Partition create failed with error")
	user := security.UserGroup{
		User:   "testuser",
		Groups: []string{},
	}
	conf := configs.PlacementRule{
		Name:  "tag",
		Value: "label1",
	}
	tr, err := newRule(conf)
	if err != nil || tr == nil {
		t.Errorf("tag rule create failed with queue name, err %v", err)
	}

	// tag does not have a value
	tags := make(map[string]string)
	appInfo := cache.NewApplicationInfo("app1", "default", "ignored", user, tags)
	var queue string
	queue, err = tr.placeApplication(appInfo, partInfo)
	if queue != "" || err != nil {
		t.Errorf("tag rule failed with no tag value '%s', err %v", queue, err)
	}

	// tag queue that exists directly in hierarchy
	tags = map[string]string{"label1": "testqueue"}
	appInfo = cache.NewApplicationInfo("app1", "default", "ignored", user, tags)
	queue, err = tr.placeApplication(appInfo, partInfo)
	if queue != "root.testqueue" || err != nil {
		t.Errorf("tag rule failed to place queue in correct queue '%s', err %v", queue, err)
	}

	// tag queue that does not exists
	tags = map[string]string{"label1": "unknown"}
	appInfo = cache.NewApplicationInfo("app1", "default", "ignored", user, tags)
	queue, err = tr.placeApplication(appInfo, partInfo)
	if queue != "" || err != nil {
		t.Errorf("tag rule placed in queue that does not exists '%s', err %v", queue, err)
	}

	// tag queue fully qualified
	tags = map[string]string{"label1": "root.testparent.testchild"}
	appInfo = cache.NewApplicationInfo("app1", "default", "ignored", user, tags)
	queue, err = tr.placeApplication(appInfo, partInfo)
	if queue != "root.testparent.testchild" || err != nil {
		t.Errorf("tag rule did fail with qualified queue '%s', error %v", queue, err)
	}

	// trying to place in a child using a parent
	conf = configs.PlacementRule{
		Name:  "tag",
		Value: "label1",
		Parent: &configs.PlacementRule{
			Name:  "tag",
			Value: "label2",
		},
	}
	tr, err = newRule(conf)
	if err != nil || tr == nil {
		t.Errorf("tag rule create failed with parent rule and qualified value, err %v", err)
	}
	tags = map[string]string{"label1": "testchild"}
	appInfo = cache.NewApplicationInfo("app1", "default", "ignored", user, tags)
	queue, err = tr.placeApplication(appInfo, partInfo)
	if queue != "" || err != nil {
		t.Errorf("tag rule with parent queue should have failed value not set '%s', error %v", queue, err)
	}
	tags = map[string]string{"label1": "testchild", "label2": "testparent"}
	appInfo = cache.NewApplicationInfo("app1", "default", "ignored", user, tags)
	queue, err = tr.placeApplication(appInfo, partInfo)
	if queue != "root.testparent.testchild" || err != nil {
		t.Errorf("tag rule with parent queue incorrect queue '%s', error %v", queue, err)
	}
}

// Create the structure for the parent rule tests
// shared by a number of rule tests
const confParentChild = `
partitions:
  - name: default
    queues:
      - name: testchild
      - name: testparent
        parent: true
`
const nameParentChild = "root.testparentnew.testchild"

func TestTagRuleParent(t *testing.T) {
	partInfo, err := CreatePartitionInfo([]byte(confParentChild))
	assert.NilError(t, err, "Partition create failed with error")
	user := security.UserGroup{
		User:   "testuser",
		Groups: []string{},
	}

	// trying to place in a child using a parent, fail to create child
	conf := configs.PlacementRule{
		Name:   "tag",
		Value:  "label1",
		Create: false,
		Parent: &configs.PlacementRule{
			Name:  "tag",
			Value: "label2",
		},
	}
	var ur rule
	ur, err = newRule(conf)
	if err != nil || ur == nil {
		t.Errorf("tag rule create failed, err %v", err)
	}

	tags := map[string]string{"label1": "testchild", "label2": "testparent"}
	appInfo := cache.NewApplicationInfo("app1", "default", "unknown", user, tags)
	var queue string
	queue, err = ur.placeApplication(appInfo, partInfo)
	if queue != "" || err != nil {
		t.Errorf("tag rule placed app in incorrect queue '%s', err %v", queue, err)
	}

	// trying to place in a child using a non creatable parent
	conf = configs.PlacementRule{
		Name:   "tag",
		Value:  "label1",
		Create: true,
		Parent: &configs.PlacementRule{
			Name:   "tag",
			Value:  "label2",
			Create: false,
		},
	}
	ur, err = newRule(conf)
	if err != nil || ur == nil {
		t.Errorf("tag rule create failed, err %v", err)
	}

	tags = map[string]string{"label1": "testchild", "label2": "testparentnew"}
	appInfo = cache.NewApplicationInfo("app1", "default", "unknown", user, tags)
	queue, err = ur.placeApplication(appInfo, partInfo)
	if queue != "" || err != nil {
		t.Errorf("tag rule placed app in incorrect queue '%s', err %v", queue, err)
	}

	// trying to place in a child using a creatable parent
	conf = configs.PlacementRule{
		Name:   "tag",
		Value:  "label1",
		Create: true,
		Parent: &configs.PlacementRule{
			Name:   "tag",
			Value:  "label2",
			Create: true,
		},
	}
	ur, err = newRule(conf)
	if err != nil || ur == nil {
		t.Errorf("tag rule create failed with queue name, err %v", err)
	}
	queue, err = ur.placeApplication(appInfo, partInfo)
	if queue != nameParentChild || err != nil {
		t.Errorf("user rule with non existing parent queue should create '%s', error %v", queue, err)
	}

	// trying to place in a child using a parent which is defined as a leaf
	conf = configs.PlacementRule{
		Name:   "tag",
		Value:  "label2",
		Create: true,
		Parent: &configs.PlacementRule{
			Name:  "tag",
			Value: "label1",
		},
	}
	ur, err = newRule(conf)
	if err != nil || ur == nil {
		t.Errorf("tag rule create failed, err %v", err)
	}

	appInfo = cache.NewApplicationInfo("app1", "default", "unknown", user, tags)
	queue, err = ur.placeApplication(appInfo, partInfo)
	if queue != "" || err == nil {
		t.Errorf("tag rule placed app in incorrect queue '%s', err %v", queue, err)
	}
}
