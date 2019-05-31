package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	dockertypes "github.com/docker/docker/api/types"
	dockerclient "github.com/docker/docker/client"
)

type BackupStruct struct {
	Networks map[string]dockertypes.NetworkCreate
	Services map[string]swarm.ServiceSpec
	Secrets  map[string]swarm.SecretSpec
	Configs  map[string]swarm.ConfigSpec
}

type Evacuation struct {
	cli    *dockerclient.Client
	Backup BackupStruct
}

func Backup(cmd *cobra.Command, args []string) {
	dest := args[0]

	err := performBackup(dest)
	if err != nil {
		panic(err)
	}
}

func performBackup(dest string) error {
	e := Evacuation{
		Backup: BackupStruct{},
	}

	cli, err := dockerclient.NewEnvClient()
	if err != nil {
		return err
	}
	e.cli = cli

	bg := context.Background()

	networks, err := cli.NetworkList(bg, dockertypes.NetworkListOptions{})
	if err != nil {
		return err
	}

	e.Backup.Networks = map[string]dockertypes.NetworkCreate{}
	networkIdNames := map[string]string{}
	skipNetworks := map[string]bool{"bridge": true, "docker_gwbridge": true, "ingress": true, "host": true, "none": true}

	for _, network := range networks {
		spec := dockertypes.NetworkCreate{
			Driver:     network.Driver,
			EnableIPv6: network.EnableIPv6,
			IPAM:       &network.IPAM,
			Internal:   network.Internal,
			Attachable: network.Attachable,
			Options:    network.Options,
			Labels:     network.Labels,
		}

		if skipNetworks[network.Name] {
			continue
		}

		networkIdNames[network.ID] = network.Name
		e.Backup.Networks[network.Name] = spec
	}

	// services
	services, err := cli.ServiceList(bg, dockertypes.ServiceListOptions{})
	if err != nil {
		return err
	}
	e.Backup.Services = map[string]swarm.ServiceSpec{}

	services, err = cli.ServiceList(bg, dockertypes.ServiceListOptions{})
	if err != nil {
		return err
	}
	e.Backup.Services = map[string]swarm.ServiceSpec{}

	for _, service := range services {
		if service.Spec.Name == "evacuation" {
			continue
		}

		serviceSpec := service.Spec
		for i, n := range serviceSpec.Networks {
			serviceSpec.Networks[i].Target = networkIdNames[n.Target]
		}

		for i, n := range serviceSpec.TaskTemplate.Networks {
			serviceSpec.TaskTemplate.Networks[i].Target = networkIdNames[n.Target]
		}

		for i, _ := range serviceSpec.TaskTemplate.ContainerSpec.Secrets {
			serviceSpec.TaskTemplate.ContainerSpec.Secrets[i].SecretID = ""
		}

		for i, _ := range serviceSpec.TaskTemplate.ContainerSpec.Configs {
			serviceSpec.TaskTemplate.ContainerSpec.Configs[i].ConfigID = ""
		}

		e.Backup.Services[serviceSpec.Name] = serviceSpec
	}

	err = e.LoadSecretsData()
	if err != nil {
		return err
	}

	bjson, err := json.MarshalIndent(e.Backup, "", " ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(dest, bjson, os.FileMode(0600))
}

func (e *Evacuation) LoadSecretsData() error {
	bg := context.Background()
	secretReferences := []*swarm.SecretReference{}

	info, err := e.cli.Info(bg)
	if err != nil {
		return err
	}

	// secrets
	e.Backup.Secrets = map[string]swarm.SecretSpec{}
	secrets, err := e.cli.SecretList(bg, dockertypes.SecretListOptions{})
	for _, secret := range secrets {
		e.Backup.Secrets[secret.Spec.Name] = secret.Spec
		secretReferences = append(secretReferences, &swarm.SecretReference{
			SecretName: secret.Spec.Name,
			SecretID:   secret.ID,
			File: &swarm.SecretReferenceFileTarget{
				Name: secret.Spec.Name,
				UID:  "0",
				GID:  "0",
				Mode: os.FileMode(020),
			},
		})
	}

	// configs
	e.Backup.Configs = map[string]swarm.ConfigSpec{}
	configs, err := e.cli.ConfigList(bg, dockertypes.ConfigListOptions{})
	for _, config := range configs {
		e.Backup.Configs[config.Spec.Name] = config.Spec
	}

	loaderSpec := swarm.ServiceSpec{
		TaskTemplate: swarm.TaskSpec{
			Placement: &swarm.Placement{
				Constraints: []string{fmt.Sprintf("node.id==%s", info.Swarm.NodeID)},
			},
			ContainerSpec: &swarm.ContainerSpec{
				Image:   "busybox",
				Command: []string{"sleep", "100000"},
				Secrets: secretReferences,
			},
		},
	}
	loaderSpec.Name = "evacuation"

	_ = e.cli.ServiceRemove(bg, "evacuation")

	service, err := e.cli.ServiceCreate(bg, loaderSpec, dockertypes.ServiceCreateOptions{})
	if err != nil {
		return err
	}

	// "com.docker.swarm.service.name"
	containerFilter := filters.NewArgs()
	containerFilter.Add("label", fmt.Sprintf("com.docker.swarm.service.id=%s", service.ID))

	var container *dockertypes.Container
	tries := 0

	for {
		if tries > 10 {
			return errors.New("failed to create export container")
		}
		containers, _ := e.cli.ContainerList(bg, dockertypes.ContainerListOptions{Filters: containerFilter})
		if len(containers) > 0 {
			container = &containers[0]
			break
		}

		tries += 1
		time.Sleep(time.Second * 5)
	}

	fmt.Printf("evac container id: %s\n", container.ID)
	for i, s := range e.Backup.Secrets {
		fmt.Printf("loading %v\n", fmt.Sprintf("/run/secrets/%s", s.Name))

		id, err := e.cli.ContainerExecCreate(bg, container.ID, dockertypes.ExecConfig{
			AttachStdout: true,
			Detach:       false,
			Tty:          true,
			Cmd:          []string{"cat", fmt.Sprintf("/run/secrets/%s", s.Name)},
		})

		if err != nil {
			return err
		}

		containerConn, err := e.cli.ContainerExecAttach(bg, id.ID, dockertypes.ExecStartCheck{Detach: false, Tty: true})
		if err != nil {
			return err
		}

		defer containerConn.Close()

		buf := new(bytes.Buffer)

		_, err = containerConn.Reader.WriteTo(buf)
		if err != nil {
			return err
		}

		containerConn.Close()

		s.Data = buf.Bytes()
		e.Backup.Secrets[i] = s
	}

	return nil
}
