#!/usr/bin/env python3
"""
End-to-end GROUP BY benchmark for the MapPool/clear() optimisation.

This is a REGRESSION CHECK, not a speedup demonstration. The optimisation saves
roughly 106ns per groupKey() call; at 3 GROUP BY phases that is ~0.3us per
document, i.e. ~0.3s per million documents -- well inside normal run-to-run
variance for a real query. Use it to confirm nothing got slower or wrong.

Usage (hybrid cluster_run setup: KV on cluster_run, query on standalone engine):

  # load data once
  ./groupby_bench.py --load --docs 200000

  # then, before and after the change
  ./groupby_bench.py --runs 20 --label before
  ./groupby_bench.py --runs 20 --label after
  ./groupby_bench.py --compare before after
"""

import argparse, json, random, statistics, sys, time, urllib.request, urllib.error

QUERY_URL = "http://127.0.0.1:9093/query/service"
BUCKET    = "default"
AUTH      = ("Administrator", "password")

CITIES     = ["San Francisco", "London", "Tokyo", "Berlin", "Sydney", "Toronto",
              "Mumbai", "Paris", "Singapore", "Austin"]
CATEGORIES = ["electronics", "grocery", "apparel", "books", "toys"]
TIERS      = ["premium", "standard", "basic"]

# Three grouping keys with ~150 distinct combinations: enough groups that the
# group-key map is built and cleared constantly, which is what we are measuring.
QUERY = f"""
SELECT   city, category, tier,
         COUNT(*)      AS n,
         AVG(price)    AS avg_price,
         SUM(quantity) AS total_qty,
         MIN(price)    AS min_price,
         MAX(price)    AS max_price
FROM     `{BUCKET}`
WHERE    price > 0
GROUP BY city, category, tier
HAVING   COUNT(*) > 1
ORDER BY n DESC
"""


def _auth_header():
    import base64
    tok = base64.b64encode(f"{AUTH[0]}:{AUTH[1]}".encode()).decode()
    return {"Authorization": "Basic " + tok}


def run_query(stmt, timeout=600):
    body = json.dumps({"statement": stmt, "timeout": f"{timeout}s"}).encode()
    hdrs = {"Content-Type": "application/json", **_auth_header()}
    req = urllib.request.Request(QUERY_URL, data=body, headers=hdrs)
    t0 = time.perf_counter()
    with urllib.request.urlopen(req, timeout=timeout + 30) as r:
        payload = json.load(r)
    wall = time.perf_counter() - t0
    if payload.get("status") != "success":
        raise RuntimeError(f"query failed: {payload.get('errors')}")
    m = payload.get("metrics", {})
    return {
        "wall_s": wall,
        "elapsed": m.get("elapsedTime"),
        "exec": m.get("executionTime"),
        "result_count": m.get("resultCount"),
    }


def load(n):
    """Load n documents via N1QL INSERT batches (no SDK dependency)."""
    print(f"loading {n} docs into `{BUCKET}` ...", flush=True)
    random.seed(42)  # deterministic dataset
    BATCH = 1000
    done = 0
    while done < n:
        vals = []
        for i in range(done, min(done + BATCH, n)):
            doc = {
                "city": random.choice(CITIES),
                "category": random.choice(CATEGORIES),
                "tier": random.choice(TIERS),
                "price": round(random.uniform(1, 1000), 2),
                "quantity": random.randint(1, 50),
                "type": "bench",
            }
            vals.append(f'("bench-{i}", {json.dumps(doc)})')
        run_query(f"INSERT INTO `{BUCKET}` (KEY, VALUE) VALUES " + ",".join(vals))
        done += BATCH
        print(f"  {done}/{n}", end="\r", flush=True)
    print(f"\nloaded {done} docs")
    run_query(f"CREATE PRIMARY INDEX IF NOT EXISTS ON `{BUCKET}`")
    print("primary index ready")


def bench(runs, warmup, label):
    print(f"warming up ({warmup} runs) ...", flush=True)
    for _ in range(warmup):
        run_query(QUERY)

    print(f"measuring ({runs} runs) ...", flush=True)
    samples = []
    for i in range(runs):
        r = run_query(QUERY)
        samples.append(r["wall_s"])
        print(f"  run {i+1:3d}: {r['wall_s']*1000:8.1f} ms  "
              f"(engine elapsed {r['elapsed']}, groups {r['result_count']})",
              flush=True)

    samples.sort()
    stats = {
        "label": label,
        "runs": runs,
        "min_ms":    samples[0] * 1000,
        "median_ms": statistics.median(samples) * 1000,
        "mean_ms":   statistics.mean(samples) * 1000,
        "stdev_ms":  (statistics.stdev(samples) * 1000) if len(samples) > 1 else 0.0,
        "p90_ms":    samples[int(len(samples) * 0.9) - 1] * 1000,
        "samples_ms": [s * 1000 for s in samples],
    }
    print(f"\n{label}: median {stats['median_ms']:.1f} ms  "
          f"mean {stats['mean_ms']:.1f} +/- {stats['stdev_ms']:.1f} ms  "
          f"min {stats['min_ms']:.1f} ms")

    fn = f"groupby_bench_{label}.json"
    with open(fn, "w") as f:
        json.dump(stats, f, indent=2)
    print(f"wrote {fn}")
    return stats


def compare(a_label, b_label):
    with open(f"groupby_bench_{a_label}.json") as f: a = json.load(f)
    with open(f"groupby_bench_{b_label}.json") as f: b = json.load(f)

    d_med = b["median_ms"] - a["median_ms"]
    pct   = d_med / a["median_ms"] * 100.0
    noise = (a["stdev_ms"] + b["stdev_ms"]) / 2.0
    noise_pct = noise / a["median_ms"] * 100.0

    print(f"\n{'':12s} {'median':>10s} {'mean':>10s} {'stdev':>10s}")
    print(f"{a_label:12s} {a['median_ms']:9.1f}ms {a['mean_ms']:9.1f}ms {a['stdev_ms']:9.1f}ms")
    print(f"{b_label:12s} {b['median_ms']:9.1f}ms {b['mean_ms']:9.1f}ms {b['stdev_ms']:9.1f}ms")
    print(f"\ndelta: {d_med:+.1f} ms ({pct:+.2f}%)   noise band: +/-{noise_pct:.2f}%")

    if abs(pct) < noise_pct:
        print("\nVERDICT: within noise -- as expected. The change is too small to")
        print("         measure end-to-end. No regression detected.")
    elif pct < 0:
        print(f"\nVERDICT: {abs(pct):.2f}% faster, outside the noise band.")
    else:
        print(f"\nVERDICT: {pct:.2f}% SLOWER -- outside noise. Investigate.")


def main():
    global QUERY_URL
    p = argparse.ArgumentParser()
    p.add_argument("--load", action="store_true", help="load the dataset first")
    p.add_argument("--docs", type=int, default=200000)
    p.add_argument("--runs", type=int, default=20)
    p.add_argument("--warmup", type=int, default=3)
    p.add_argument("--label", default="run")
    p.add_argument("--compare", nargs=2, metavar=("BEFORE", "AFTER"))
    p.add_argument("--query-url", default=None)
    args = p.parse_args()

    if args.query_url:
        QUERY_URL = args.query_url

    if args.compare:
        compare(*args.compare)
        return

    try:
        run_query("SELECT 1")
    except (urllib.error.URLError, OSError) as e:
        sys.exit(f"cannot reach query engine at {QUERY_URL}: {e}\n"
                 f"start cluster_run and cbq-engine first.")

    if args.load:
        load(args.docs)

    bench(args.runs, args.warmup, args.label)


if __name__ == "__main__":
    main()
