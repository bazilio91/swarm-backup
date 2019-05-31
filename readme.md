swarm backup
====

Creates backup of services, secrets, configs, networks into json file with possibility to restore into different docker swarm cluster.

By default uses `DOCKER_HOST` and other envs (just like docker/cli)

Usage: 

```
swarm-backup backup b.json
swarm-backup restore b.json
```

Notes:
---

Internal networks, such as:
- bridge
- docker_gwbridge
- ingress

are not backing up.

Should be run on active manager.

Backup process creates new service `evacuation` to get secrets data on the current swarm manager.
