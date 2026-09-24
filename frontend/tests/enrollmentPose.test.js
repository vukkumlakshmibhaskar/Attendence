import test from "node:test";
import assert from "node:assert/strict";
import { evaluateEnrollmentPose, rememberPoseMetrics, isPoseHistoryStable, validateEnrollmentDetection } from "../src/utils/enrollmentPose.js";

const front = { yawScore: -0.07, pitchScore: -0.30, roll: 0.02 };
const box = { left: 0.40, top: 0.25, width: 0.22, height: 0.40 };
const sample = (overrides = {}) => ({ ...front, ...overrides });

test("screenshot regression: a steady front face at pitch -0.30 can calibrate", () => {
  assert.equal(evaluateEnrollmentPose("front", front, {}).valid, true);
});

test("all five poses progress from the person's measured neutral position", () => {
  const captures = {};
  const steps = [
    ["front", front],
    ["left", sample({ yawScore: 0.15 })],
    ["right", sample({ yawScore: -0.29 })],
    ["look_up", sample({ pitchScore: -0.46 })],
    ["look_down", sample({ pitchScore: -0.14 })],
  ];
  for (const [pose, metrics] of steps) {
    assert.equal(evaluateEnrollmentPose(pose, metrics, captures).valid, true, pose);
    captures[pose] = metrics;
  }
  assert.equal(Object.keys(captures).length, 5);
});

test("holding the same face pose cannot fill every step", () => {
  for (const pose of ["left", "right", "look_up", "look_down"]) {
    assert.equal(evaluateEnrollmentPose(pose, front, { front }).valid, false, pose);
  }
});

test("opposite turns and excessive head tilt are rejected", () => {
  for (const [pose, metrics] of [
    ["left", sample({ yawScore: -0.4 })],
    ["right", sample({ yawScore: 0.3 })],
    ["look_up", sample({ pitchScore: 0.0 })],
    ["look_down", sample({ pitchScore: -0.6 })],
    ["front", sample({ roll: 0.5 })],
    ["front", sample({ yawScore: 0.5 })],
    ["front", sample({ pitchScore: NaN })],
  ]) assert.equal(evaluateEnrollmentPose(pose, metrics, { front }).valid, false, pose);
  assert.equal(evaluateEnrollmentPose("left", sample({ yawScore: 0.3 }), {}).valid, false);
});

test("camera pitch offsets are calibrated separately for each person", () => {
  for (const pitch of [-0.45, -0.30, -0.10, 0.05]) {
    const neutral = sample({ pitchScore: pitch });
    assert.equal(evaluateEnrollmentPose("front", neutral, {}).valid, true);
    assert.equal(evaluateEnrollmentPose("look_up", sample({ pitchScore: pitch - 0.16 }), { front: neutral }).valid, true);
    assert.equal(evaluateEnrollmentPose("look_down", sample({ pitchScore: pitch + 0.16 }), { front: neutral }).valid, true);
  }
});

test("stability needs several samples and resets on a new pose", () => {
  const history = { current: [] }, pose = { current: "" };
  for (let i = 0; i < 3; i++) {
    rememberPoseMetrics(history, pose, "front", { box, metrics: sample({ pitchScore: -0.30 + i * 0.01 }) });
    assert.equal(isPoseHistoryStable(history.current), i >= 2);
  }
  rememberPoseMetrics(history, pose, "left", { box, metrics: sample({ yawScore: 0.3 }) });
  assert.equal(isPoseHistoryStable(history.current), false);
});

test("moving faces cannot trigger automatic capture", () => {
  const history = { current: [] }, pose = { current: "" };
  for (const yawScore of [-0.4, 0.1, 0.5]) {
    rememberPoseMetrics(history, pose, "front", { box, metrics: sample({ yawScore }) });
  }
  assert.equal(isPoseHistoryStable(history.current), false);
});

test("capture requires one fresh detected face; errors are actionable", () => {
  const fresh = { count: 1, box, sampledAt: 10000 };
  assert.equal(validateEnrollmentDetection(fresh, 10500).valid, true);
  assert.equal(validateEnrollmentDetection(fresh, 12000).valid, false);
  assert.equal(validateEnrollmentDetection({ ...fresh, count: 2 }, 10500).valid, false);
  assert.equal(validateEnrollmentDetection({ ...fresh, count: 0, box: null }, 10500).valid, false);
  assert.equal(validateEnrollmentDetection({ ...fresh, error: "Connection failed" }, 10500).message, "Connection failed");
});
