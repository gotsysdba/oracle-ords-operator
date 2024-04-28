# Frequently Asked Questions

## **Q: I have `spec.autoRestart` set to `false`, why do my pods cycle on changes?**
The `spec.autoRestart` is honoured when settings within existing, configured pools in `spec.poolSettings` are modified.
There are several scenarios in which the pods will always restart, despite the `autoRestart` specification:

  * `spec.workloadType` is changed
  * `spec.imageName` is changed
  * `spec.globalSettings.standaloneHttpPort` is changed
  * Pools are added or removed from `spec.poolSettings`

---

