package cmd

import (
  "context"
  "encoding/json"
  "io/ioutil"

  "github.com/docker/docker/api/types"
  "github.com/spf13/cobra"

  dockerclient "github.com/docker/docker/client"
)

func Restore(cmd *cobra.Command, args []string) {
  src := args[0]

  err := performRestore(src)
  if err != nil {
    panic(err)
  }
}

func performRestore(src string) error {
  b, err := ioutil.ReadFile(src)
  if err != nil {
    return err
  }

  e := Evacuation{}
  cli, err := dockerclient.NewEnvClient()
  if err != nil {
    return err
  }
  e.cli = cli

  bg := context.Background()

  err = json.Unmarshal(b, &e.Backup)
  if err != nil {
    return err
  }

  for name, net := range e.Backup.Networks {
    _, err := e.cli.NetworkCreate(bg, name, net)
    if err != nil {
      return err
    }
  }

  secretNameId := map[string]string{}
  for n, s := range e.Backup.Secrets {
    r, err := e.cli.SecretCreate(bg, s)
    if err != nil {
      return err
    }

    secretNameId[n] = r.ID
  }

  configNameId := map[string]string{}
  for n, c := range e.Backup.Configs {
    r, err := e.cli.ConfigCreate(bg, c)
    if err != nil {
      return err
    }

    configNameId[n] = r.ID
  }

  for _, service := range e.Backup.Services {
    for is, secret := range service.TaskTemplate.ContainerSpec.Secrets {
      service.TaskTemplate.ContainerSpec.Secrets[is].SecretID = secretNameId[secret.SecretName]
    }
    for ic, config := range service.TaskTemplate.ContainerSpec.Configs {
      service.TaskTemplate.ContainerSpec.Configs[ic].ConfigID = configNameId[config.ConfigName]
    }

    _, err = e.cli.ServiceCreate(bg, service, types.ServiceCreateOptions{})
    if err != nil {
      return err
    }
  }

  return nil
}
