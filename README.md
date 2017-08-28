# Istio rudder proxy

The easiset way to install rudder and rudder proxy is to inject both containers into tiller-deploy
deployment. I prepared `patch.sh` that will correctly update tiller-deploy.

```
helm init
./patch.sh
```

By default `patch.sh` will use 'istio/proxy_debug:0.2.0' and 'istio/proxy_init:0.2.0' containers.
Please overwrite `--tag` argument that is passed to istio-rudder-proxy container if you need
to use another version.

Istio proxy is a small wrapper over istioctl kube-inject command. Most of the arguments are
intentionally copied from istioctl. 
I highly recommend to check out [istioctl kube-inject documentation](https://istio.io/docs/reference/commands/istioctl.html#istioctl-kube-inject) for more details.

# How to disable istio proxy injection?

If you want to disable proxy injection for certain objects use annotation that is passed to
istio-rudder-proxy binary. By defaul it is 'istio.skip'. Then this annotation can be used as in the
following example:

```
kind: Job
metadata:
  name: test-job
  annotations:
    istio.skip: 1
```
