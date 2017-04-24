package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/stephenrlouie/travis/model"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

var (
	ErrServiceNotRunning     = errors.New("Service Not Running.")
	ErrServiceAlreadyRunning = errors.New("Service Already Running.")
	ErrServiceNotFound       = errors.New("Service Not Found.")
	ErrUnsupportedDriverType = errors.New("Unsupported Driver Type.")
	ErrNilServiceOperation   = errors.New("Nil ServiceOperation")
	ErrDriverUnavailable     = errors.New("Driver is Unavailable.")
	ErrUnableToGetState      = errors.New("Unable to get container state.")
	ErrMalformedID           = errors.New("Malformed service ID.")
)

var (
	SaveStateKey     = "task_save_state"
	SaveDir          = "/var/lib/sigma"
	OutputDirDefault = "/var/lib/sigma"
)

type DockerTasker struct {
	Client    *client.Client
	outputDir string
}

func NewDockerTasker() (*DockerTasker, error) {
	t := &DockerTasker{}
	var err error
	// Alternatively, EnvClient(host, version, blah...)?
	//t.Client, err = client.NewEnvClient()
	var cl *http.Client
	t.Client, err = client.NewClient(client.DefaultDockerHost, "1.24", cl, nil)
	if err != nil {
		return nil, err
	}

	os.MkdirAll(SaveDir, 0777)

	if _, err = os.Stat(SaveDir); err != nil {
		return nil, err
	}

	t.outputDir = SaveDir
	if _, err := t.Client.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("Unable to connect to docker: %v", err)
	}
	return t, nil
}

func (d *DockerTasker) Available() bool {
	_, err := d.Client.Ping(context.Background())
	return err == nil
}

func (d *DockerTasker) Run(so *model.ServiceOperation) error {
	if so == nil || so.Service.Id == "" {
		return ErrNilServiceOperation
	}
	var env []string
	if e, ok := so.Plugin.Config["env"]; ok {
		if earr, ok2 := e.([]string); ok2 {
			env = earr
		}
	}
	cmd := []string{}
	if so.Operation != "" {
		cmd = append(cmd, so.Operation)
	}

	cfg2 := &container.Config{
		Image: so.Plugin.Image,
		Cmd:   cmd,
		Tty:   true,
		Env:   env,
	}

	dir := path.Join(SaveDir, so.Service.Id)
	if err := os.MkdirAll(dir, 0777); err != nil {
		return err
	}
	jInput, err := json.Marshal(model.ServiceDataItemsToMap(so.Service.Input))
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(path.Join(dir, "input"), jInput, 0666); err != nil {
		return err
	}

	hostConfig := &container.HostConfig{
		Binds: []string{fmt.Sprintf("%s:%s", dir, OutputDirDefault)},
	}

	if err = d.pull(context.Background(), so.Plugin.Image, ""); err != nil {
		return err
	}
	_, err = d.Client.ContainerCreate(context.Background(), cfg2, hostConfig, nil, so.Service.Id)
	if err != nil {
		return err
	}
	err = d.Client.ContainerStart(context.Background(), so.Service.Id, types.ContainerStartOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (d *DockerTasker) Remove(so *model.ServiceOperation) error {
	if so == nil || so.Service.Id == "" {
		return ErrNilServiceOperation
	}
	// IDS cannot contain `.`,`/`,`~`
	if strings.ContainsAny(so.Service.Id, "./~") {
		return ErrMalformedID
	}
	defer os.RemoveAll(path.Join(d.outputDir, so.Service.Id))
	e := d.Client.ContainerRemove(context.Background(), so.Service.Id, types.ContainerRemoveOptions{
		Force: true,
	})
	if e != nil {
		return e
	}
	return nil
}

func (d *DockerTasker) Stop(so *model.ServiceOperation, timeout time.Duration) error {
	if so == nil || so.Service.Id == "" {
		return ErrNilServiceOperation
	}
	return d.Client.ContainerStop(context.Background(), so.Service.Id, &timeout)
}

func (d *DockerTasker) Progress(so *model.ServiceOperation) (string, error) {
	if so == nil || so.Service.Id == "" {
		return "", ErrNilServiceOperation
	}
	b, err := ioutil.ReadFile(path.Join(d.outputDir, so.Service.Id, "progress"))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (d *DockerTasker) Logs(so *model.ServiceOperation) (string, error) {
	if so == nil {
		return "", ErrNilServiceOperation
	}

	out, err := d.Client.ContainerLogs(context.Background(), so.Service.Id, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return "", err
	}
	b, err := d.read(context.Background(), out, out.Close)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// We assume exitCode = 0 --> success.
func (d *DockerTasker) Status(so *model.ServiceOperation) (string, error) {
	if so == nil || so.Service.Id == "" {
		return "", ErrNilServiceOperation
	}

	r, err := d.Client.ContainerInspect(context.Background(), so.Service.Id)
	if err != nil {
		return "", err
	}
	if r.State == nil {
		return "", ErrUnableToGetState
	}

	if r.State.Status == "created" {
		return model.ServiceOperationStatusWaiting, nil
	} else if r.State.Dead {
		return model.ServiceOperationStatusStopped, nil
	} else if r.State.Running {
		return model.ServiceOperationStatusRunning, nil
	} else if r.State.Status == "exited" {
		if r.State.ExitCode == 137 { // sigKILL, docker=128 + sh 9 = 137
			return model.ServiceOperationStatusStopped, nil
		} else if r.State.ExitCode == 0 {
			return model.ServiceOperationStatusFinished, nil
		}
		return model.ServiceOperationStatusFailed, nil
	}
	return model.ServiceOperationStatusUnknown, nil
}

func (d *DockerTasker) Outputs(so *model.ServiceOperation) (map[string][]string, error) {
	if so == nil || so.Service.Id == "" {
		return nil, ErrNilServiceOperation
	}

	b, err := ioutil.ReadFile(path.Join(d.outputDir, so.Service.Id, "output"))
	if err != nil {
		return nil, err
	}
	var outs map[string][]string
	if err = json.Unmarshal(b, &outs); err != nil {
		return nil, err
	}
	return outs, nil
}

func (d *DockerTasker) pull(ctx context.Context, image string, creds string) error {
	resp, err := d.Client.ImagePull(ctx, image, types.ImagePullOptions{
		RegistryAuth: creds,
	})
	if err != nil {
		return err
	}
	pullOutput, readErr := d.read(ctx, resp, resp.Close)
	if readErr != nil {
		return readErr
	}
	re := regexp.MustCompile("\"error\"\\s?:\\s?\"(.*)\"")
	if re == nil {
		return errors.New(string(pullOutput))
	}
	errsOutput := re.FindSubmatch(pullOutput)
	if len(errsOutput) > 1 {
		return errors.New(string(errsOutput[1]))
	}
	return readErr
}

func (d *DockerTasker) read(ctx context.Context, reader io.ReadCloser, closer func() error) ([]byte, error) {
	done := make(chan bool)
	var containerOutput []byte
	var readErr error
	go func() {
		containerOutput, readErr = readAll(reader)
		done <- true
	}()
	closed := false
	for {
		select {
		case <-done:
			if !closed {
				closer()
				closed = true
			}
			return containerOutput, readErr
		case <-ctx.Done():
			if !closed {
				closer()
				closed = true
			}
		}
	}
}

// readAll takes in a bufio.Reader and reads in the bytes, in chunks.
func readAll(reader io.Reader) ([]byte, error) {
	var b []byte
	buf := make([]byte, 1024)
	for {
		// read a chunk
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			return b, err
		}
		if n == 0 {
			break
		}
		b = append(b, buf[:n]...)
	}
	return b, nil
}
