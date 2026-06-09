#!/usr/bin/env bash

run_fake_session_checks() {
  local codex_session sessions_output events_output session_output restore_plan_output completion_values
  local before_restore_lines after_restore_lines

  info "fake smoke: inspect events and sessions"
  events_output=$(run_adp "$REPO_ROOT" events list --workspace game-a --task "$task_id" --type run_finished --limit 2)
  assert_contains "$events_output" "run_finished" "events list output"
  assert_contains "$events_output" "codex" "events list output"
  assert_contains "$events_output" "claude" "events list output"
  assert_contains "$events_output" "$task_id" "events list output"

  codex_session=$(session_id_by_agent "$events_file" codex)
  if [ -z "$codex_session" ]; then
    cat "$events_file" >&2
    fail "codex session id missing in event log"
  fi

  sessions_output=$(run_adp "$REPO_ROOT" sessions list --workspace game-a --agent codex --task "$task_id")
  assert_contains "$sessions_output" "$codex_session" "sessions list output"
  assert_contains "$sessions_output" "codex" "sessions list output"
  assert_contains "$sessions_output" "$task_id" "sessions list output"

  completion_values=$(run_adp "$REPO_ROOT" completion values sessions --workspace game-a)
  assert_contains "$completion_values" "$codex_session" "completion session values output"

  events_output=$(run_adp "$REPO_ROOT" events list --session "$codex_session" --task "$task_id" --limit 2)
  assert_contains "$events_output" "$codex_session" "events list session output"
  assert_contains "$events_output" "$task_id" "events list session output"
  assert_contains "$events_output" "codex" "events list session output"
  assert_contains "$events_output" "run_started" "events list session output"
  assert_contains "$events_output" "run_finished" "events list session output"

  session_output=$(run_adp "$REPO_ROOT" sessions show "$codex_session")
  assert_contains "$session_output" "session_id: $codex_session" "sessions show output"
  assert_contains "$session_output" "task_id: $task_id" "sessions show output"
  assert_contains "$session_output" "run_started" "sessions show output"
  assert_contains "$session_output" "run_finished" "sessions show output"

  before_restore_lines=$(line_count "$events_file")
  restore_plan_output=$(run_adp "$REPO_ROOT" sessions restore-plan "$codex_session")
  assert_contains "$restore_plan_output" "session_id: $codex_session" "sessions restore-plan output"
  assert_contains "$restore_plan_output" "status: ready" "sessions restore-plan output"
  assert_contains "$restore_plan_output" "suggested_command:" "sessions restore-plan output"
  assert_contains "$restore_plan_output" "adp run codex --workspace game-a" "sessions restore-plan output"
  assert_contains "$restore_plan_output" "--task $task_id" "sessions restore-plan output"
  assert_contains "$restore_plan_output" "-- --probe codex-payload" "sessions restore-plan output"
  assert_contains "$restore_plan_output" "missing_fields: -" "sessions restore-plan output"
  after_restore_lines=$(line_count "$events_file")
  if [ "$after_restore_lines" != "$before_restore_lines" ]; then
    cat "$events_file" >&2
    fail "sessions restore-plan appended events: before=$before_restore_lines after=$after_restore_lines"
  fi
}
