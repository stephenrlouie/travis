package task

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cto-github.cisco.com/GSP/mu-sigma/model"
)

func testServiceOperation(image string, driverConfig map[string]interface{}) *model.ServiceOperation {
	return &model.ServiceOperation{
		Plugin: model.Plugin{
			Config: driverConfig,
			Image:  image,
		},
		Service: model.Service{
			Id: "Test" + model.GenerateUUID(),
		},
	}
}

func TestMain(m *testing.M) {
	SaveDir = "/tmp/sigma/tests"
	rc := m.Run()
	os.RemoveAll(SaveDir)
	os.Exit(rc)

}

func TestRemove(t *testing.T) {
	tasker, err := NewDockerTasker()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	so := testServiceOperation("mikeynap/sigma:latest", map[string]interface{}{
		"env": []string{"COMMAND=echo blah; echo blahblah >> /var/lib/sigma/progress; echo blahblahblah > /var/lib/sigma/progress; echo blah2"},
	})

	idsErr := map[string]error{
		"test":        nil,
		"../test2234": ErrMalformedID,
		"~/test23":    ErrMalformedID,
		"/test23":     ErrMalformedID,
	}

	for k := range idsErr {
		s := *so
		s.Service.Id = k
		tasker.Run(&s)
	}

	time.Sleep(1 * time.Second)

	for k, er := range idsErr {
		s := *so
		s.Service.Id = k
		if err := tasker.Remove(&s); err != er {
			t.Errorf("Remove should have returned %v, got %v instead ", er, err)
		}
	}
}

func TestDockerStatus(t *testing.T) {
	tasker, err := NewDockerTasker()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	tests := map[*model.ServiceOperation]string{
		testServiceOperation("mikeynap/sigma:latest", map[string]interface{}{
			"env": []string{"COMMAND=exit 0"},
		}): model.ServiceOperationStatusFinished,
		testServiceOperation("mikeynap/sigma:latest", map[string]interface{}{
			"env": []string{"COMMAND=exit 1"},
		}): model.ServiceOperationStatusFailed,
		testServiceOperation("mikeynap/sigma:latest", map[string]interface{}{
			"env": []string{"COMMAND=sleep 1000"},
		}): model.ServiceOperationStatusStopped,
	}

	for so, _ := range tests {
		tasker.Run(so)
	}
	oneIsRunning := false
	time.Sleep(1 * time.Second)
	for so, o := range tests {
		if s, _ := tasker.Status(so); s == model.ServiceOperationStatusRunning {
			oneIsRunning = true
		}
		tasker.Stop(so, 1*time.Second)
		if s, err := tasker.Status(so); err != nil || s != o {
			t.Errorf("Err: %v, got %v, expected %v", err, s, o)
		}
	}
	if !oneIsRunning {
		t.Error("Tasks never returned Running")
	}
}

func TestDocker(t *testing.T) {
	tasker, err := NewDockerTasker()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	so := testServiceOperation("mikeynap/sigma:latest", map[string]interface{}{
		"env": []string{"COMMAND=echo blah; echo blahblah >> /var/lib/sigma/progress;  sleep 3; echo blahblahblah > /var/lib/sigma/progress; echo blah2"},
	})
	t.Log(tasker.Run(so))
	time.Sleep(2000 * time.Millisecond)
	if l, e := tasker.Logs(so); e != nil || strings.TrimSpace(l) != "blah" {
		t.Errorf("Incorrect Logs (or error %v), received %v, expected 'blah'", e, l)
	}
	if s, e := tasker.Progress(so); e != nil || strings.TrimSpace(s) != "blahblah" {
		t.Errorf("Incorrect Progress (or error %v), received %v, expected 'blahblah'", e, s)
	}

	if s, e := tasker.Status(so); e != nil || s != model.ServiceOperationStatusRunning {
		t.Errorf("Incorrect Status (or error %v), received %v, expected %v", e, s, model.ServiceOperationStatusRunning)
	}
	time.Sleep(5 * time.Second)

	if l, e := tasker.Logs(so); e != nil || strings.TrimSpace(l) != "blah\r\nblah2" {
		t.Errorf("Incorrect Logs (or error %v), received %v, expected 'blah'", e, l)
	}
	if s, e := tasker.Progress(so); e != nil || strings.TrimSpace(s) != "blahblahblah" {
		t.Errorf("Incorrect Progress (or error %v), received %v, expected 'blahblahblah'", e, s)
	}
	if s, e := tasker.Status(so); e != nil || s != model.ServiceOperationStatusFinished {
		t.Errorf("Incorrect Status (or error %v), received %v, expected %v", e, s, model.ServiceOperationStatusFinished)
	}
	tasker.Remove(so)
}

func TestDockerCleanup(t *testing.T) {
	tasker, err := NewDockerTasker()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	so := testServiceOperation("mikeynap/sigma:latest", map[string]interface{}{
		"env": []string{"COMMAND=echo blah; echo startingsleep >> /var/lib/sigma/progress;  sleep 100"},
	})
	tasker.Run(so)
	time.Sleep(2 * time.Second)
	if err := tasker.Remove(so); err != nil {
		t.Errorf("Got Error %v when calling tasker.REmove()", err)
	}
	time.Sleep(2 * time.Second)

	if _, err := tasker.Logs(so); err == nil {
		t.Error("Should not be able to get logs, container should be removed.")
	}
	if _, e := tasker.Status(so); e == nil {
		t.Error("Should not be able to get status, container should be removed")
	}
}

func TestDockerOutputs(t *testing.T) {
	outputs := map[string][]string{
		"key1": []string{"value1"},
		"out":  []string{"put"},
	}

	b, err := json.Marshal(outputs)
	if err != nil {
		fmt.Println("err: ", err)
	}
	so := testServiceOperation("mikeynap/sigma:latest", map[string]interface{}{
		"env": []string{fmt.Sprintf("COMMAND=echo blah; echo '%v' >> /var/lib/sigma/output", string(b))},
	})

	tasker, err := NewDockerTasker()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	err = tasker.Run(so)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	time.Sleep(1500 * time.Millisecond)
	ops, err := tasker.Outputs(so)
	t.Log(ops)
	if err != nil || len(ops) != 2 || ops["key1"][0] != outputs["key1"][0] || ops["out"][0] != outputs["out"][0] {
		t.Errorf("Error (%v) or len == %d, should == 2", err, len(ops))
	}
	tasker.Remove(so)
}

func TestResume(t *testing.T) {
	so := testServiceOperation("mikeynap/sigma:latest", map[string]interface{}{
		"env": []string{"COMMAND=echo blah; echo blahhhhhh >> /var/lib/sigma/output; sleep 120"},
	})
	tasker, err := NewDockerTasker()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	err = tasker.Run(so)
	if err != nil {
		t.Error("Error: ", err)
		t.FailNow()
	}

	tasker2, err := NewDockerTasker()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	time.Sleep(2 * time.Second)

	logs1, err1 := tasker.Logs(so)
	logs2, err2 := tasker2.Logs(so)

	if logs1 != "blah\r\n" || logs1 != logs2 || err1 != nil || err2 != nil {
		t.Error("Resuming Task should have same logs as running task. ", logs1, logs2, err1, err2)
	}
	tasker.Remove(so)

}
