# HPA Autoscaling

The controller-manager manages the autoscaling of pods across the cluster. 
It uses the following parameters to manage the cooldown/delays.

- --horizontal-pod-autoscaler-downscale-delay (default 5m0s)
- --horizontal-pod-autoscaler-upscale-delay (default 3m0s)

For example if you want to override these values you can do the following:
```yaml
experimental:
  horizontalPodAutoscalerDownscaleDelay: 1m0s
  horizontalPodAutoscalerUpscaleDelay: 30s
```