## 1. Update API Server Deployment

- [ ] 1.1 Add hostPath volume for `/opt/local-path-provisioner/` to the apiserver Deployment template (type: Directory, readOnly: true)
- [ ] 1.2 Add corresponding volumeMount to the apiserver container at `/opt/local-path-provisioner/`
- [ ] 1.3 Verify with `helm template` that the volume and mount render correctly

## 2. Rebuild Docker Images

- [ ] 2.1 Rebuild the controlplane image from current source
- [ ] 2.2 Rebuild the sidecar image from current source (includes log tee + trace collection)
- [ ] 2.3 Rebuild the hydration image from current source (includes `.devcontainer` + `.aot` directory generation)
- [ ] 2.4 Rebuild the agent-base image from current source

## 3. Import Images into k0s

- [ ] 3.1 Import controlplane image into k0s (`k0s ctr images import`)
- [ ] 3.2 Import sidecar image into k0s
- [ ] 3.3 Import hydration image into k0s
- [ ] 3.4 Import agent-base image into k0s

## 4. Restart Deployments

- [ ] 4.1 Rollout restart the apiserver deployment
- [ ] 4.2 Rollout restart the controller deployment
- [ ] 4.3 Rollout restart any other affected deployments (Temporal worker, etc.)
- [ ] 4.4 Verify all pods are Running and Ready

## 5. Add `deploy:all` Taskfile Task

- [ ] 5.1 Add `deploy:all` task to Taskfile that chains: build all images, import all images, restart all deployments
- [ ] 5.2 Verify `task deploy:all` completes without errors

## 6. End-to-End Verification: Logs

- [ ] 6.1 Create a test run via the web UI or API
- [ ] 6.2 Open the Logs tab while the run is executing
- [ ] 6.3 Verify log lines appear in real-time as the agent produces output
- [ ] 6.4 After the run completes, verify historical logs are still accessible

## 7. End-to-End Verification: Files

- [ ] 7.1 After a run completes, open the Files tab
- [ ] 7.2 Verify the workspace directory tree is displayed
- [ ] 7.3 Verify individual files can be opened and their contents viewed

## 8. End-to-End Verification: Shell

- [ ] 8.1 For a running agent, open a shell session
- [ ] 8.2 Verify interactive terminal works (run a command, see output)
- [ ] 8.3 For a completed run, start a Debug Run and verify terminal access

## 9. End-to-End Verification: Traces

- [ ] 9.1 After a run completes, open the traces view
- [ ] 9.2 Verify trace spans are displayed showing the agent's execution timeline

## 10. Bug Fixes

- [ ] 10.1 Fix any code bugs discovered during end-to-end verification
- [ ] 10.2 Rebuild and redeploy affected images after fixes
- [ ] 10.3 Re-verify the affected features work
