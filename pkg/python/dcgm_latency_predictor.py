#!/usr/bin/env python3

import argparse
import csv
import json
import os
import sys


DEFAULT_CSV_PATH = "/data/cyq/cpn_controller/pkg/python/results.csv"


def resource_path(filename):
    if getattr(sys, "frozen", False) and hasattr(sys, "_MEIPASS"):
        return os.path.join(sys._MEIPASS, filename)
    return os.path.join(os.path.dirname(os.path.abspath(__file__)), filename)


def clamp(value, lower, upper):
    return max(lower, min(upper, value))


def normalize_metric(value):
    metric = float(value)
    if metric < 0:
        metric = 0.0
    if metric > 1.2:
        metric = metric / 100.0
    return clamp(metric, 0.0, 1.0)


def resolve_csv_path(explicit_path):
    candidates = []
    if explicit_path:
        candidates.append(explicit_path)
    candidates.append(resource_path("results.csv"))
    candidates.append(DEFAULT_CSV_PATH)
    candidates.append(os.path.join(os.getcwd(), "pkg", "python", "results.csv"))

    seen = set()
    for candidate in candidates:
        if not candidate:
            continue
        normalized = os.path.abspath(candidate)
        if normalized in seen:
            continue
        seen.add(normalized)
        if os.path.exists(normalized):
            return normalized
    raise FileNotFoundError("results.csv not found")


def load_baselines(csv_path):
    baselines = {}
    with open(csv_path, "r", newline="") as handle:
        reader = csv.DictReader(handle)
        for row in reader:
            model_name = row["model_name"].strip()
            latency_ms = float(row["measure_avg_latency_ms"])
            baselines[model_name] = latency_ms
            if model_name.endswith(".onnx"):
                baselines[model_name[:-5]] = latency_ms
    return baselines


def latency_factor(sm_active, sm_occupancy, dram_active):
    sm = normalize_metric(sm_active)
    occ = normalize_metric(sm_occupancy)
    dram = normalize_metric(dram_active)

    # High SM/occupancy indicates the GPU is already busy.
    # When SM pressure is high and DRAM is not low, latency should rise sharply.
    pressure = 0.50 * sm + 0.35 * occ + 0.15 * dram
    factor = 1.0

    if pressure <= 0.35:
        factor += pressure * 0.10
    elif pressure <= 0.55:
        factor += 0.035 + (pressure - 0.35) * 1.20
    elif pressure <= 0.75:
        factor += 0.275 + (pressure - 0.55) * 2.80
    else:
        factor += 0.715 + (pressure - 0.75) * 2.00

    compute_pressure = 0.55 * sm + 0.45 * occ
    factor += max(0.0, compute_pressure - 0.55) * 1.35
    factor += max(0.0, sm - 0.60) * 0.70
    factor += max(0.0, occ - 0.55) * 0.50

    if sm >= 0.55 and occ >= 0.50:
        factor += 0.14 + max(0.0, compute_pressure - 0.58) * 0.80

    memory_gap = dram - compute_pressure
    factor += max(0.0, memory_gap - 0.05) * 0.30

    if sm < 0.30 and occ < 0.30:
        factor *= 0.92

    return clamp(factor, 0.90, 1.95), {
        "sm_active": sm,
        "sm_occupancy": occ,
        "dram_active": dram,
        "pressure": round(pressure, 6),
    }


def predict_latency_ms(model_name, baselines, sm_active, sm_occupancy, dram_active):
    if model_name not in baselines:
        raise KeyError("unknown model name: {0}".format(model_name))

    base_latency_ms = baselines[model_name]
    factor, normalized_dcgm = latency_factor(sm_active, sm_occupancy, dram_active)
    predicted_latency_ms = base_latency_ms * factor

    return {
        "model_name": model_name,
        "baseline_latency_ms": round(base_latency_ms, 6),
        "predicted_latency_ms": round(predicted_latency_ms, 6),
        "predicted_epoch_time_s": round(predicted_latency_ms / 1000.0, 6),
        "latency_factor": round(factor, 6),
        "dcgm": normalized_dcgm,
    }


def build_parser():
    parser = argparse.ArgumentParser(
        description="Predict model latency from a baseline CSV and realtime DCGM metrics."
    )
    parser.add_argument("model_name", help="Standard model name, for example vgg19_bs128_224x224")
    parser.add_argument("--cuda-cores", type=float, default=0.0, help="Accepted for interface compatibility")
    parser.add_argument("--gpu-core-frequency", type=float, default=0.0, help="Accepted for interface compatibility")
    parser.add_argument("--memory-bandwidth", type=float, default=0.0, help="Accepted for interface compatibility")
    parser.add_argument("--compute-benchmark-time", type=float, default=0.0, help="Accepted for interface compatibility")
    parser.add_argument("--memory-benchmark-time", type=float, default=0.0, help="Accepted for interface compatibility")
    parser.add_argument("--sm-active", required=True, type=float, help="Supports either 0-1 or 0-100 input")
    parser.add_argument("--sm-occupancy", required=True, type=float, help="Supports either 0-1 or 0-100 input")
    parser.add_argument("--dram-active", required=True, type=float, help="Supports either 0-1 or 0-100 input")
    parser.add_argument("--csv-path", default="", help="Optional results.csv override")
    parser.add_argument("--json", action="store_true", help="Print structured output instead of a plain number")
    parser.add_argument("--print-ms", action="store_true", help="Print predicted latency in milliseconds")
    return parser


def main():
    parser = build_parser()
    args = parser.parse_args()

    try:
        csv_path = resolve_csv_path(args.csv_path)
        baselines = load_baselines(csv_path)
        prediction = predict_latency_ms(
            args.model_name,
            baselines,
            args.sm_active,
            args.sm_occupancy,
            args.dram_active,
        )
    except Exception as exc:
        sys.stderr.write("{0}\n".format(exc))
        return 1

    if args.json:
        sys.stdout.write(json.dumps(prediction, sort_keys=True) + "\n")
        return 0

    if args.print_ms:
        sys.stdout.write("{0:.6f}\n".format(prediction["predicted_latency_ms"]))
        return 0

    sys.stdout.write("{0:.6f}\n".format(prediction["predicted_epoch_time_s"]))
    return 0


if __name__ == "__main__":
    sys.exit(main())
