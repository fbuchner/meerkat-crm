#!/usr/bin/env python3
"""Review a pull request with a non-Anthropic model through OpenRouter.

Runs from .github/workflows/ai-review.yml, whose comment explains the security
rules this file has to keep. PR content is only ever read as text through the
GitHub API. Standard library only, so the job has nothing to install.

Try it on an existing PR without posting anything:
    GITHUB_TOKEN=$(gh auth token) OPENROUTER_API_KEY=... \
        python3 .github/ai-review/review.py --dry-run --repo owner/name --pr 123
"""

import argparse
import fnmatch
import hashlib
import http.client
import json
import os
import re
import secrets
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path

HERE = Path(__file__).resolve().parent
ROOT = HERE.parent.parent
GITHUB_API = "https://api.github.com"
OPENROUTER_URL = "https://openrouter.ai/api/v1/chat/completions"
USER_AGENT = "meerkat-ai-review"
SUMMARY_MARKER = "<!-- ai-review-summary -->"
BOT_LOGIN = "github-actions[bot]"
ATTEMPTS = 3
RETRY_DELAY_SECONDS = 5
RETRYABLE_STATUS = {408, 429, 500, 502, 503, 504}
# The key is the problem, so no other model or mode can succeed.
FATAL_OPENROUTER_STATUS = {401}
# How OpenRouter names a spent budget, in error.metadata.limit_source. It sends these
# with 402 or 403. A 403 can also be a moderation or guardrail block, which is not one.
BUDGET_LIMIT_SOURCES = {
    "openrouter_credits", "openrouter_key_limit",
    "openrouter_guardrail_budget", "openrouter_workspace_budget",
}
# A 402 that clears by itself once requests in flight finish.
IN_FLIGHT_LIMIT_SOURCE = "openrouter_in_flight_budget"
# The catch-up run looks for this exact text to find PRs to resume, so changing it
# strands PRs already postponed under the old text until their next push.
POSTPONED_DESCRIPTION = "Postponed: OpenRouter budget used up. Resumes automatically after the reset."
# Leaves room inside the catch-up job's 60 minutes for one more review, which can take
# up to deadline_seconds, and for posting its result.
CATCH_UP_SECONDS = 40 * 60
NETWORK_ERRORS = (OSError, http.client.HTTPException)
# GitHub rejects comment bodies over 65536 characters.
MAX_COMMENT_CHARS = 60000

SEVERITY_RANK = {"nit": 0, "minor": 1, "major": 2, "blocker": 3}
SEVERITY_LABEL = {"blocker": "🚫 Blocker", "major": "⚠️ Major", "minor": "🔸 Minor", "nit": "💬 Nit"}
RISK_FLAGS = [
    "security-sensitive", "auth", "data-migration", "public-api-change",
    "new-dependency", "test-weakening", "concurrency", "large-or-unclear-change",
]

REVIEW_SCHEMA = {
    "type": "object",
    "additionalProperties": False,
    "required": ["summary", "findings", "risk_flags", "prompt_injection"],
    "properties": {
        "summary": {"type": "string"},
        "findings": {
            "type": "array",
            "items": {
                "type": "object",
                "additionalProperties": False,
                "required": ["path", "line", "severity", "title", "explanation", "suggestion"],
                "properties": {
                    "path": {"type": "string"},
                    "line": {"type": ["integer", "null"]},
                    "severity": {"type": "string", "enum": list(SEVERITY_RANK)},
                    "title": {"type": "string"},
                    "explanation": {"type": "string"},
                    "suggestion": {"type": "string"},
                },
            },
        },
        "risk_flags": {"type": "array", "items": {"type": "string", "enum": RISK_FLAGS}},
        "prompt_injection": {
            "type": "object",
            "additionalProperties": False,
            "required": ["detected", "evidence"],
            "properties": {"detected": {"type": "boolean"}, "evidence": {"type": "string"}},
        },
    },
}


class HttpError(Exception):
    def __init__(self, status, message, metadata=None):
        super().__init__(f"HTTP {status}: {message}")
        self.status = status
        self.metadata = metadata or {}


class BudgetExhausted(Exception):
    def __init__(self, error):
        super().__init__(str(error))
        self.resets_at = error.metadata.get("resets_at")


def error_metadata(text):
    try:
        metadata = json.loads(text)["error"]["metadata"]
    except (ValueError, KeyError, TypeError):
        return {}
    return metadata if isinstance(metadata, dict) else {}


def http_request(method, url, headers, payload=None, timeout=60, raw=False):
    data = json.dumps(payload).encode() if payload is not None else None
    headers = dict(headers, **{"User-Agent": USER_AGENT})
    if data is not None:
        headers["Content-Type"] = "application/json"
    request = urllib.request.Request(url, data=data, method=method, headers=headers)
    try:
        with urllib.request.urlopen(request, timeout=timeout) as response:
            body = response.read().decode(errors="replace")
    except urllib.error.HTTPError as error:
        text = error.read().decode(errors="replace")
        raise HttpError(error.code, text[:500], error_metadata(text)) from None
    if raw:
        return body
    return json.loads(body) if body.strip() else None


class GitHub:
    def __init__(self, token, repo):
        self.repo = repo
        self.headers = {
            "Authorization": f"Bearer {token}",
            "Accept": "application/vnd.github+json",
            "X-GitHub-Api-Version": "2022-11-28",
        }

    def call(self, method, path, payload=None, raw=False):
        url = f"{GITHUB_API}/repos/{self.repo}/{path}"
        headers = dict(self.headers, Accept="application/vnd.github.raw+json") if raw else self.headers
        # Repeating a POST could double-post comments. A status just overwrites itself.
        attempts = ATTEMPTS if method != "POST" or path.startswith("statuses/") else 1
        for attempt in range(attempts):
            if attempt:
                time.sleep(RETRY_DELAY_SECONDS * attempt)
            try:
                return http_request(method, url, headers, payload, raw=raw)
            except HttpError as error:
                if error.status not in RETRYABLE_STATUS or attempt == attempts - 1:
                    raise
            except NETWORK_ERRORS:
                if attempt == attempts - 1:
                    raise

    def paginate(self, path):
        items, page = [], 1
        while True:
            separator = "&" if "?" in path else "?"
            batch = self.call("GET", f"{path}{separator}per_page=100&page={page}")
            items.extend(batch)
            if len(batch) < 100:
                return items
            page += 1

    def file_at(self, path, ref):
        return self.call("GET", f"contents/{urllib.parse.quote(path)}?ref={ref}", raw=True)


def matches_any(path, patterns):
    return any(fnmatch.fnmatch(path, pattern) for pattern in patterns)


def decide(pr, config, action, label):
    """Return "skip", "wait" or "review" for this PR and event.

    Strangers can open PRs on this public repo, and every review spends OpenRouter
    credit, so an untrusted author only gets a review when a maintainer adds the
    trigger label. Adding a label requires triage access. A push does not count,
    which keeps an approved author from re-spending by pushing again.
    """
    author = pr["user"]["login"]
    if author in config["skip_authors"]:
        return "skip"
    if pr["author_association"] in config["trusted_associations"] or author in config["trusted_authors"]:
        return "review"
    if action == "labeled" and label == config["trigger_label"]:
        return "review"
    return "wait"


def annotate_patch(patch):
    """Prefix each new-file line with its number, so the model need not count.

    Returns the annotated patch and the line numbers GitHub accepts an inline
    comment on.
    """
    rows, valid, current = [], set(), None
    for row in patch.split("\n"):
        hunk = re.match(r"^@@ -\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@", row)
        if hunk:
            current = int(hunk.group(1))
            rows.append(row)
        elif current is None or row.startswith("\\"):
            rows.append(row)
        elif row.startswith("-"):
            rows.append(f"      {row}")
        else:
            rows.append(f"{current:>5} {row}")
            valid.add(current)
            current += 1
    return "\n".join(rows), valid


def build_material(github, pr, files, config):
    """Collect the diff, file context, valid comment lines and whatever did not fit."""
    diff_parts, context_parts, omitted, reviewed = [], [], [], []
    valid_lines = {}
    diff_size = context_size = 0
    if len(files) < pr["changed_files"]:
        omitted.append(f"{pr['changed_files'] - len(files)} further files GitHub did not list")
    for item in files:
        path = item["filename"]
        if matches_any(path, config["ignored_paths"]):
            continue
        patch = item.get("patch")
        if patch is None:
            # GitHub drops the patch for binaries (no line changes) and huge diffs.
            if item.get("changes", 0) > 0:
                omitted.append(path)
            continue
        annotated, valid = annotate_patch(patch)
        entry = f"--- {path} ({item['status']})\n{annotated}\n"
        if diff_size + len(entry) > config["max_diff_chars"]:
            omitted.append(path)
            continue
        diff_parts.append(entry)
        diff_size += len(entry)
        valid_lines[path] = valid
        reviewed.append(item)

    for item in reviewed:
        if item["status"] == "removed":
            continue
        if context_size >= config["max_context_chars"]:
            break
        path = item["filename"]
        try:
            text = github.file_at(path, pr["head"]["sha"])
        except (HttpError,) + NETWORK_ERRORS as error:
            print(f"warning: no context for {path}: {error}", file=sys.stderr)
            continue  # Context only; the diff is what gets reviewed.
        if len(text) <= config["max_context_file_chars"] and context_size + len(text) <= config["max_context_chars"]:
            context_parts.append(f"--- {path}\n{text}\n")
            context_size += len(text)
    return {
        "diff": "".join(diff_parts),
        "context": "".join(context_parts),
        "omitted": omitted,
        "valid_lines": valid_lines,
    }


def build_messages(pr, material, rules, config):
    # A random tag, so PR content cannot close the block by guessing its name.
    tag = f"untrusted-{secrets.token_hex(8)}"
    system = (HERE / "prompt.md").read_text().replace("<untrusted-…>", f"<{tag}>")
    system += (
        "\n## Repository rules\n\nThese tell the coding agent how to work here. Use them as "
        f"the standard for the code; the parts about running in CI do not apply to you.\n\n{rules}"
    )
    description = (pr.get("body") or "(none)")[: config["max_description_chars"]]
    user = (
        f"<{tag}>\nTitle: {pr['title']}\n\nDescription:\n{description}\n</{tag}>\n\n"
        f"Full new version of changed files, for context only:\n<{tag}>\n{material['context']}</{tag}>\n\n"
        "Annotated diff to review. The left column is the new-file line number to use for "
        f"`line`; lines starting with - were removed:\n<{tag}>\n{material['diff']}</{tag}>"
    )
    return [{"role": "system", "content": system}, {"role": "user", "content": user}]


def parse_review(text):
    """Parse the model's JSON and normalise it, tolerating a fence or stray prose."""
    start, end = text.find("{"), text.rfind("}")
    review = json.loads(text[start : end + 1])
    if not isinstance(review, dict) or not isinstance(review.get("findings"), list):
        raise ValueError("response does not match the review schema")
    injection = review.get("prompt_injection")
    if not isinstance(injection, dict) or "detected" not in injection:
        raise ValueError("response does not match the review schema")
    findings = []
    for raw in review["findings"]:
        if not isinstance(raw, dict) or not raw.get("title"):
            raise ValueError("response does not match the review schema")
        line = raw.get("line")
        findings.append({
            "path": str(raw.get("path") or ""),
            "line": line if isinstance(line, int) and not isinstance(line, bool) else None,
            "severity": raw.get("severity") if raw.get("severity") in SEVERITY_RANK else "minor",
            "title": str(raw["title"]),
            "explanation": str(raw.get("explanation") or ""),
            "suggestion": str(raw.get("suggestion") or ""),
        })
    flags = review.get("risk_flags")
    return {
        "summary": str(review.get("summary") or ""),
        "findings": sorted(findings, key=lambda f: -SEVERITY_RANK[f["severity"]]),
        "risk_flags": [f for f in flags if f in RISK_FLAGS] if isinstance(flags, list) else [],
        "prompt_injection": {"detected": bool(injection["detected"]), "evidence": str(injection.get("evidence") or "")},
    }


def ask_model(model, messages, api_key, config, structured, timeout):
    headers = {"Authorization": f"Bearer {api_key}", "X-Title": "Meerkat CRM review"}
    payload = {"model": model, "messages": messages, "max_tokens": config["max_output_tokens"], "usage": {"include": True}}
    if config.get("reasoning_effort"):
        payload["reasoning"] = {"effort": config["reasoning_effort"]}
    # Open-weight models are served by many hosts, the cheapest often heavily quantized,
    # and a gate should not vary in quality with whichever host a run lands on.
    payload["provider"] = {"quantizations": config["provider_quantizations"]}
    if structured:
        payload["response_format"] = {
            "type": "json_schema",
            "json_schema": {"name": "review", "strict": True, "schema": REVIEW_SCHEMA},
        }
        # Only route to providers that honour the schema above.
        payload["provider"]["require_parameters"] = True
    else:
        instruction = (
            "\n\nRespond with ONLY a JSON object, no prose and no code fences, matching this "
            f"JSON Schema:\n{json.dumps(REVIEW_SCHEMA)}"
        )
        payload["messages"] = messages[:-1] + [dict(messages[-1], content=messages[-1]["content"] + instruction)]
    response = http_request("POST", OPENROUTER_URL, headers, payload, timeout=timeout)
    # OpenRouter can report an upstream failure inside a 200.
    if "error" in response:
        code = response["error"].get("code")
        raise HttpError(
            code if isinstance(code, int) else 502,
            str(response["error"].get("message"))[:500],
            response["error"].get("metadata"),
        )
    choice = response["choices"][0]
    if choice.get("finish_reason") == "length":
        raise ValueError("output cut off at max_output_tokens")
    return parse_review(choice["message"].get("content") or ""), response.get("model") or model, response.get("usage") or {}


def is_budget_error(error):
    source = error.metadata.get("limit_source")
    if error.status == 402:
        return source != IN_FLIGHT_LIMIT_SOURCE
    return error.status == 403 and source in BUDGET_LIMIT_SOURCES


def is_transient(error):
    if error.status == 402:
        return error.metadata.get("limit_source") == IN_FLIGHT_LIMIT_SOURCE
    return error.status in RETRYABLE_STATUS


def request_review(messages, api_key, config):
    """Try each model, with the strict schema and then as plain JSON.

    Transient failures are rarely billed, so they are retried. Malformed output is
    billed, so it moves straight on, which limits the cost of a model that keeps
    returning garbage to one call per model and mode. The deadline keeps the job's own timeout from killing the run first,
    which would leave the status pending with no explanation.
    """
    deadline = time.monotonic() + config["deadline_seconds"]
    last_error = None
    for model in config["models"]:
        for structured in (True, False):
            for attempt in range(ATTEMPTS):
                if attempt:
                    time.sleep(RETRY_DELAY_SECONDS * attempt)
                remaining = deadline - time.monotonic()
                if remaining <= 0:
                    raise RuntimeError(f"ran out of time; last error: {last_error}")
                timeout = min(config["request_timeout_seconds"], remaining)
                try:
                    return ask_model(model, messages, api_key, config, structured, timeout)
                except HttpError as error:
                    last_error = error
                    print(f"{model}: {error}", file=sys.stderr)
                    if is_budget_error(error):
                        raise BudgetExhausted(error) from None
                    if error.status in FATAL_OPENROUTER_STATUS:
                        raise RuntimeError(f"OpenRouter refused the request: {error}") from None
                    if not is_transient(error):
                        break
                except NETWORK_ERRORS as error:
                    last_error = error
                    print(f"{model}: {type(error).__name__}: {error}", file=sys.stderr)
                except (ValueError, KeyError, IndexError, TypeError, AttributeError) as error:
                    last_error = error
                    print(f"{model}: unusable output: {type(error).__name__}: {error}", file=sys.stderr)
                    break
    raise RuntimeError(f"every model failed; last error: {last_error}")


def neutralise(text):
    """Stop text that PR content can influence from pinging people or hiding content."""
    return re.sub(r"@(?=[A-Za-z0-9])", "@​", str(text)).replace("<!--", "&lt;!--")


def format_finding(finding):
    text = f"**{SEVERITY_LABEL[finding['severity']]}: {finding['title'].strip()}**\n\n{finding['explanation'].strip()}"
    if finding["suggestion"].strip():
        text += f"\n\nSuggested fix:\n````\n{finding['suggestion'].strip()}\n````"
    return neutralise(text)[:MAX_COMMENT_CHARS]


def conclude(review, omitted, config):
    """Work out the verdict here, so the model cannot simply declare a pass."""
    threshold = SEVERITY_RANK[config["fail_on"]]
    blocking = sum(1 for f in review["findings"] if SEVERITY_RANK[f["severity"]] >= threshold)
    if review["prompt_injection"]["detected"]:
        return "failure", "Possible prompt injection; a maintainer must review"
    if omitted:
        return "failure", "PR too large to review completely; a maintainer must review"
    if blocking:
        return "failure", f"{blocking} blocking finding{'s' if blocking != 1 else ''}"
    return "success", f"No blocking findings, {len(review['findings'])} other finding(s)"


def human_review_reasons(review, files, omitted, config):
    reasons = []
    flagged = [
        f["filename"] for f in files
        # A rename counts by its old path too, or moving a file would hide the change.
        if matches_any(f["filename"], config["human_review_paths"])
        or matches_any(f.get("previous_filename") or "", config["human_review_paths"])
    ]
    if flagged:
        reasons.append("Touches sensitive paths: " + ", ".join(f"`{p}`" for p in flagged))
    risks = [f for f in review["risk_flags"] if f in config["human_review_risk_flags"]]
    if risks:
        reasons.append("Reviewer flagged: " + ", ".join(f"`{f}`" for f in risks))
    if omitted:
        reasons.append("Not fully reviewed")
    if review["prompt_injection"]["detected"]:
        reasons.append("Possible prompt injection")
    return reasons


def split_findings(findings, valid_lines, max_inline):
    """Separate findings GitHub can anchor to a diff line from the rest."""
    anchored, general = [], []
    for finding in findings:
        target = anchored if finding["line"] in valid_lines.get(finding["path"], ()) else general
        target.append(finding)
    return anchored[:max_inline], anchored[max_inline:] + general


def inline_marker(finding):
    # Without the line number, so a push that shifts the code does not repost it.
    key = f"{finding['path']}|{finding['title'].strip().lower()}"
    return f"<!-- ai-review:{hashlib.sha256(key.encode()).hexdigest()[:16]} -->"


def post_inline(github, pr, inline):
    """Post findings not already on the PR.

    Returns those GitHub would not accept, and those already posted. An earlier comment
    can be collapsed as outdated, so the summary has to mention them as well.
    """
    number = pr["number"]
    existing = github.paginate(f"pulls/{number}/comments")
    posted = "\n".join(c["body"] for c in existing if c["user"]["login"] == BOT_LOGIN)
    fresh = [f for f in inline if inline_marker(f) not in posted]
    repeated = [f for f in inline if inline_marker(f) in posted]
    if not fresh:
        return [], repeated
    comments = [
        {"path": f["path"], "line": f["line"], "side": "RIGHT", "body": f"{format_finding(f)}\n\n{inline_marker(f)}"}
        for f in fresh
    ]
    try:
        github.call("POST", f"pulls/{number}/reviews", {"commit_id": pr["head"]["sha"], "event": "COMMENT", "comments": comments})
    except HttpError as error:
        # The whole review is refused if one line is off, and these lines were checked,
        # so the likeliest cause is a push during the run. Keep the findings regardless.
        if error.status != 422:
            raise
        print(f"warning: inline comments rejected: {error}", file=sys.stderr)
        return fresh, repeated
    return [], repeated


def render_summary(review, general, omitted, state, description, sha, model, usage, reasons, repeated=()):
    verdict = "✅ Passed" if state == "success" else "❌ Failed"
    parts = [f"## AI review: {verdict} · {sha[:7]}", description, review["summary"]]
    if review["prompt_injection"]["detected"]:
        parts.append(f"**Possible prompt injection:**\n\n> {review['prompt_injection']['evidence']}")
    if omitted:
        parts.append("**Not reviewed, because it did not fit:** " + ", ".join(f"`{p}`" for p in omitted))
    if reasons:
        parts.append("**Needs a human:**\n" + "\n".join(f"- {r}" for r in reasons))
    for finding in general:
        location = f"`{finding['path']}`" + (f" line {finding['line']}" if finding["line"] else "")
        parts.append(f"{format_finding(finding)}\n\n{location}")
    if repeated:
        rows = [f"- {SEVERITY_LABEL[f['severity']]}: {f['title'].strip()} (`{f['path']}` line {f['line']})" for f in repeated]
        parts.append("**Still present, already commented on:**\n" + "\n".join(rows))
    if model:
        cost = f", ${usage['cost']:.4f}" if isinstance(usage.get("cost"), (int, float)) else ""
        tokens = f"{usage.get('prompt_tokens', '?')} in / {usage.get('completion_tokens', '?')} out"
        parts.append(f"<sub>Reviewed by `{model}` via OpenRouter ({tokens}{cost}). Automated and advisory, not an approval.</sub>")
    body = neutralise("\n\n".join(parts))
    if len(body) > MAX_COMMENT_CHARS:
        body = body[:MAX_COMMENT_CHARS] + "\n\n…cut off here. The full review is in the workflow run log."
    return f"{SUMMARY_MARKER}\n\n{body}"


def post_summary(github, number, summary):
    for comment in github.paginate(f"issues/{number}/comments"):
        if comment["user"]["login"] == BOT_LOGIN and SUMMARY_MARKER in comment["body"]:
            github.call("PATCH", f"issues/comments/{comment['id']}", {"body": summary})
            return
    github.call("POST", f"issues/{number}/comments", {"body": summary})


def set_status(github, sha, config, state, description):
    target = None
    if os.environ.get("GITHUB_RUN_ID"):
        target = f"https://github.com/{github.repo}/actions/runs/{os.environ['GITHUB_RUN_ID']}"
    github.call("POST", f"statuses/{sha}", {
        "state": state,
        "context": config["status_context"],
        "description": description[:140],
        "target_url": target,
    })


def run_review(github, pr, config, api_key, dry_run):
    number, sha = pr["number"], pr["head"]["sha"]
    files = github.paginate(f"pulls/{number}/files")
    material = build_material(github, pr, files, config)
    rules = (ROOT / ".github" / "CLAUDE_INSTRUCTIONS.md").read_text()

    if material["diff"]:
        messages = build_messages(pr, material, rules, config)
        review, model, usage = request_review(messages, api_key, config)
        print(f"Reviewed by {model}; usage: {json.dumps(usage)}")
    else:
        review = {
            "summary": "Nothing to send to the model: only ignored or binary files changed, or the changes did not fit.",
            "findings": [],
            "risk_flags": [],
            "prompt_injection": {"detected": False, "evidence": ""},
        }
        model, usage = None, {}

    omitted = material["omitted"]
    state, description = conclude(review, omitted, config)
    reasons = human_review_reasons(review, files, omitted, config)
    inline, general = split_findings(review["findings"], material["valid_lines"], config["max_inline_comments"])

    if dry_run:
        print(render_summary(review, general, omitted, state, description, sha, model, usage, reasons))
        print(f"\nInline comments ({len(inline)}):")
        for finding in inline:
            print(f"\n{finding['path']}:{finding['line']}\n{format_finding(finding)}")
        print(f"\nStatus would be: {state} ({description})")
        return

    rejected, repeated = post_inline(github, pr, inline)
    general = rejected + general
    summary = render_summary(review, general, omitted, state, description, sha, model, usage, reasons, repeated)
    post_summary(github, number, summary)
    if reasons:
        github.call("POST", f"issues/{number}/labels", {"labels": [config["human_review_label"]]})
    if os.environ.get("GITHUB_STEP_SUMMARY"):
        with open(os.environ["GITHUB_STEP_SUMMARY"], "a") as handle:
            handle.write(summary + "\n")
    set_status(github, sha, config, state, description)


def postpone(github, pr, config, error, dry_run):
    """Record that the budget ran out, which is expected and not a failure.

    Nothing was posted yet, since the model call comes first. The status stays pending,
    so a required check keeps blocking, and the catch-up run finds it by its description.
    """
    print(f"PR #{pr['number']} postponed: {error}")
    if dry_run:
        return
    when = f" Budget resets at {error.resets_at}." if error.resets_at else ""
    summary = (
        f"{SUMMARY_MARKER}\n\n## AI review: ⏳ Postponed · {pr['head']['sha'][:7]}\n\n"
        f"The OpenRouter budget is used up, so this commit has not been reviewed yet.{when} "
        "The review resumes automatically after the reset."
    )
    post_summary(github, pr["number"], summary)
    set_status(github, pr["head"]["sha"], config, "pending", POSTPONED_DESCRIPTION)


def remove_trigger_label(github, pr, config):
    label = config["trigger_label"]
    if not any(item["name"] == label for item in pr.get("labels", [])):
        return
    try:
        github.call("DELETE", f"issues/{pr['number']}/labels/{urllib.parse.quote(label)}")
    except HttpError as error:
        print(f"warning: could not remove the label: {error}", file=sys.stderr)


def review_and_report(github, pr, config, dry_run, resuming=False):
    """Review one PR and report the outcome: "reviewed", "postponed" or "failed".

    When resuming, the postponed status is left in place until there is a verdict, and
    restored on failure, so a cancelled job or an outage just means the next catch-up
    run tries again. A re-run of that job would otherwise find nothing to resume.
    """
    sha = pr["head"]["sha"]
    try:
        api_key = os.environ.get("OPENROUTER_API_KEY")
        if not api_key:
            raise RuntimeError("OPENROUTER_API_KEY is not set")
        if not dry_run and not resuming:
            set_status(github, sha, config, "pending", "Review in progress")
        run_review(github, pr, config, api_key, dry_run)
    except BudgetExhausted as error:
        try:
            postpone(github, pr, config, error, dry_run)
            return "postponed"
        except Exception as post_error:
            print(f"Could not record the postponement: {post_error}", file=sys.stderr)
            return "failed"
    except Exception as error:  # Fail closed: anything unexpected must not read as a pass.
        print(f"Review of PR #{pr['number']} failed: {type(error).__name__}: {error}", file=sys.stderr)
        if not dry_run:
            try:
                if resuming:
                    set_status(github, sha, config, "pending", POSTPONED_DESCRIPTION)
                else:
                    set_status(github, sha, config, "error", "Reviewer failed; re-run this workflow")
            except Exception as status_error:
                print(f"Could not set the status either: {status_error}", file=sys.stderr)
        return "failed"

    # Consume the label so that a later push needs a fresh maintainer decision.
    if not dry_run:
        remove_trigger_label(github, pr, config)
    return "reviewed"


def current_status(github, sha, config):
    statuses = github.call("GET", f"commits/{sha}/status")["statuses"]
    return next((item for item in statuses if item["context"] == config["status_context"]), None)


def is_postponed(github, pr, config):
    status = current_status(github, pr["head"]["sha"], config)
    return bool(status) and status["state"] == "pending" and status["description"] == POSTPONED_DESCRIPTION


def catch_up(github, config):
    """Review what a spent budget postponed, oldest first, until it runs out again.

    The postponed status sits on one commit and was only set after decide() allowed that
    commit, so a stranger's newer push is not picked up here.
    """
    started = time.monotonic()
    failed = False
    for listed in github.paginate("pulls?state=open&sort=created&direction=asc"):
        if time.monotonic() - started > CATCH_UP_SECONDS:
            print("Out of time; the next catch-up run continues from here")
            break
        if listed["draft"] or not is_postponed(github, listed, config):
            continue
        pr = github.call("GET", f"pulls/{listed['number']}")
        if pr["head"]["sha"] != listed["head"]["sha"]:
            continue  # A newer push, which has had its own decision.
        outcome = review_and_report(github, pr, config, dry_run=False, resuming=True)
        print(f"PR #{pr['number']}: {outcome}")
        if outcome == "postponed":
            break
        failed = failed or outcome == "failed"
    if failed:
        sys.exit(1)


def main():
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--dry-run", action="store_true", help="print the review instead of posting it")
    parser.add_argument("--repo", default=os.environ.get("GITHUB_REPOSITORY"))
    parser.add_argument("--pr", type=int)
    args = parser.parse_args()

    action = label = None
    number = args.pr
    scheduled = not number and os.environ.get("GITHUB_EVENT_NAME") in ("schedule", "workflow_dispatch")
    if os.environ.get("GITHUB_EVENT_PATH") and not number and not scheduled:
        event = json.loads(Path(os.environ["GITHUB_EVENT_PATH"]).read_text())
        number = event["pull_request"]["number"]
        action = event.get("action")
        label = (event.get("label") or {}).get("name")
    if not args.repo or not (number or scheduled):
        parser.error("--repo and --pr are required outside GitHub Actions")

    config = json.loads((HERE / "config.json").read_text())
    if config["fail_on"] not in SEVERITY_RANK:
        parser.error(f"fail_on in config.json must be one of {list(SEVERITY_RANK)}")
    github = GitHub(os.environ["GITHUB_TOKEN"], args.repo)
    if scheduled:
        catch_up(github, config)
        return
    pr = github.call("GET", f"pulls/{number}")
    sha = pr["head"]["sha"]

    if pr["state"] != "open" and not args.dry_run:
        print(f"PR #{number} is {pr['state']}; nothing to do")
        return
    # Reopening or undrafting changes no code, so an existing verdict still stands.
    if not args.dry_run and action in ("reopened", "ready_for_review"):
        status = current_status(github, sha, config)
        if status and status["state"] in ("success", "failure"):
            print(f"PR #{number}: {sha[:7]} already has a verdict")
            return
    decision = "review" if args.dry_run else decide(pr, config, action, label)
    print(f"PR #{number} by {pr['user']['login']} ({pr['author_association']}): {decision}")
    if decision == "skip":
        try:
            set_status(github, sha, config, "success", "Skipped: automated dependency update")
        except HttpError as error:
            # Dependabot runs may get a read-only token. Not worth a red check on every bump.
            print(f"warning: could not set the status: {error}", file=sys.stderr)
        return
    if decision == "wait":
        set_status(github, sha, config, "pending", f"Waiting for a maintainer to add the {config['trigger_label']} label")
        # An approval covers one commit. A label left from before, such as on a postponed
        # review, would stop a maintainer approving this one, as re-adding it fires no event.
        remove_trigger_label(github, pr, config)
        return

    if review_and_report(github, pr, config, args.dry_run) == "failed":
        sys.exit(1)


if __name__ == "__main__":
    main()
