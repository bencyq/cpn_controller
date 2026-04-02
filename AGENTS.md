# Repository Guidelines

## Project Structure & Module Organization
`cmd/` contains runnable entrypoints: `controller` is the main scheduler, while `affinity`, `k8ssche`, `mbf`, and `rr` are strategy variants. Core scheduling logic lives in `pkg/controller/` with sample topology JSON files (`example*.json`), model baselines, and the default `yaml_queue/`. Supporting packages include `pkg/utils/` for CSV and queue helpers, `pkg/python/` for the socket-based predictor and training scripts, and `pkg/yaml_template/` for job templates. Treat `test_logs/`, `metrics`, and `nohup.out` as experiment artifacts, and avoid editing `vendor/` unless you are intentionally upgrading dependencies.

## Build, Test, and Development Commands
Use Go 1.23.x as declared in `go.mod`.

- `go build ./...` compiles all packages and binaries; this is the safest default verification step.
- `go run ./cmd/controller` starts the main controller against the JSON, Prometheus, and kubeconfig paths referenced from `pkg/controller/example*.json`.
- `go run ./cmd/rr` runs the round-robin baseline; swap in other `cmd/*` packages to test alternative strategies.
- `go test ./pkg/controller -run 'TestPrepareJobForNodeUsesHamiUUID|TestApplyLoadBalanceFactor'` runs fast unit-style checks.
- `go test ./...` is not a safe default here: several tests call `main()` or `NewMonitor()` and may talk to real clusters.
- `python3 pkg/python/random_forest_train.py` regenerates the ignored `*.pt` predictor weights when needed.

## Coding Style & Naming Conventions
Format Go code with `gofmt`; let it control tabs and spacing. Use lower-case package names, `CamelCase` for exported Go identifiers, and keep strategy-specific code under matching package names such as `pkg/rr` or `pkg/mbf`. Preserve the existing zero-padded YAML queue naming (`001.yaml`) and keep config fixtures in `pkg/controller/`.

## Testing Guidelines
Prefer pure unit tests in `*_test.go` beside the package under test. For Kubernetes behavior, favor object-level assertions or fake clients over tests that construct a full `Monitor`. If a change depends on lab infrastructure, document the required `example*.json`, Prometheus reachability, and kubeconfig setup in the PR.

## Commit & Pull Request Guidelines
Recent history uses short, imperative subjects, in both Chinese and English, such as `完成beta因子的测试`, `Add fifo`, and `fix some bugs`. Keep commits focused on one change. PRs should state the touched strategy or controller path, summarize config or YAML changes, list exact verification commands, and attach relevant logs or metric snapshots when scheduler behavior changes.

## Security & Configuration Tips
Do not commit lab-specific `kubeconfig`, trained `*.pt` weights, or cache files; they are already ignored. Review any edits to `pkg/controller/example*.json`, socket paths in `pkg/python/`, and HAMI or GPU resource settings carefully before merging.
