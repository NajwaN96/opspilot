#!/usr/bin/env bash
# Read-only account, free-tier, and portfolio resource report. Prints no credentials.
set -euo pipefail

python3 - << 'PY'
import json, subprocess

def aws(*args):
    result = subprocess.run(["aws", "--region", "us-east-1", "--output", "json", *args], capture_output=True, text=True)
    if result.returncode != 0:
        raise SystemExit(result.stderr.strip().splitlines()[-1][:240])
    return json.loads(result.stdout or "{}")

plan = aws("freetier", "get-account-plan-state")
plan.pop("accountId", None)
usage = aws("freetier", "get-free-tier-usage")
identity = aws("sts", "get-caller-identity")
account = identity.get("Account", "")
masked = "********" + account[-4:] if len(account) == 12 else "unknown"

functions = aws("lambda", "list-functions").get("Functions") or []
names = sorted(item.get("FunctionName", "") for item in functions)
urls = []
if "opspilot-portfolio" in names:
    try:
        url = aws("lambda", "get-function-url-config", "--function-name", "opspilot-portfolio")
        urls.append({"function": "opspilot-portfolio", "auth": url.get("AuthType"), "url": url.get("FunctionUrl")})
    except SystemExit:
        urls.append({"function": "opspilot-portfolio", "url": "unreadable"})

logs = aws("logs", "describe-log-groups", "--log-group-name-prefix", "/aws/lambda/opspilot-portfolio").get("logGroups") or []
log_rows = [{"name": item.get("logGroupName"), "retentionDays": item.get("retentionInDays")} for item in logs]

report = {
    "accountMasked": masked,
    "region": "us-east-1",
    "accountPlanType": plan.get("accountPlanType"),
    "accountPlanStatus": plan.get("accountPlanStatus"),
    "remainingCredits": plan.get("accountPlanRemainingCredits"),
    "planExpiration": plan.get("accountPlanExpirationDate"),
    "freeTierUsageRecords": len(usage.get("freeTierUsages") or []),
    "freeTierUsages": usage.get("freeTierUsages") or [],
    "lambdaFunctions": names,
    "functionUrls": urls,
    "logGroups": log_rows,
    "note": "Built-in credit emails are not spending caps. This script does not enable Cost Explorer.",
}
print(json.dumps(report, indent=2))
PY
