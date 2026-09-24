import test from "node:test";
import assert from "node:assert/strict";
import { evaluateEnrollmentFrame, rememberPoseMetrics, isPoseHistoryStable } from "../src/utils/enrollmentPose.js";

const now = 10000;
const front = { yawScore: -0.07, pitchScore: -0.30, roll: 0.02 };
const detection = (box = {}) => ({
  count: 1, sampledAt: now - 100, frameSize: { width: 640, height: 360 },
  box: { x: 328, y: 110, width: 112, height: 144, ...box },
});
const analysis = (metrics = {}, box = { left: 0.82, top: 0.4, width: 0.22, height: 0.4 }) => ({
  faceDetected: true, valid: true, box, metrics: { ...front, yawScore: 0.20, ...metrics },
});
const evaluate = (pose, measured = analysis(), detected = detection(), time = now) =>
  evaluateEnrollmentFrame(pose, measured, detected, { front }, time);

test("visible side face passes despite drift in the landmark-estimated box", () => {
  const result = evaluate("left");
  assert.equal(result.valid, true);
  assert.equal(result.box.left, 328 / 640);
  assert.equal(result.box.width, 112 / 640);
});

test("both side poses can be off-centre while fully inside the camera", () => {
  for (const x of [30, 510]) {
    assert.equal(evaluate("left", analysis(), detection({ x })).valid, true);
    assert.equal(evaluate("right", analysis({ yawScore: -0.30 }), detection({ x })).valid, true);
  }
});

test("narrower side profiles are accepted when they retain enough detail", () => {
  assert.equal(evaluate("left", analysis(), detection({ width: 55, height: 135 })).valid, true);
  assert.equal(evaluate("left", analysis(), detection({ width: 25, height: 60 })).valid, false);
});

test("cut-off faces are rejected at each image edge", () => {
  for (const box of [{ x: 0 }, { y: 0 }, { x: 528 }, { y: 216 }]) {
    const result = evaluate("left", analysis(), detection(box));
    assert.equal(result.valid, false);
    assert.match(result.message, /whole face stays in view/);
  }
});

test("position never substitutes for a genuine turn in the requested direction", () => {
  for (const x of [30, 328, 510]) {
    assert.equal(evaluate("left", analysis({ yawScore: front.yawScore }), detection({ x })).valid, false);
    assert.equal(evaluate("left", analysis({ yawScore: -0.30 }), detection({ x })).valid, false);
    assert.equal(evaluate("right", analysis(), detection({ x })).valid, false);
  }
});

test("fresh single-face detection and visible landmarks remain mandatory", () => {
  for (const detected of [
    { ...detection(), count: 0, box: null },
    { ...detection(), count: 2 },
    { ...detection(), sampledAt: now - 2000 },
    { ...detection(), frameSize: null },
    detection({ x: NaN }),
  ]) assert.equal(evaluate("left", analysis(), detected).valid, false);
  const missing = evaluate("left", { valid: false, faceDetected: false, metrics: null });
  assert.equal(missing.valid, false);
  assert.match(missing.message, /Turn slightly back/);
});

test("stability uses the actual detected face location rather than the drifting estimated box", () => {
  const history = { current: [] }, pose = { current: "" };
  for (const left of [0.35, 0.82, 0.56, 0.85]) {
    const result = evaluate("left", analysis({}, { left, top: 0.4, width: 0.2, height: 0.4 }));
    rememberPoseMetrics(history, pose, "left", result);
  }
  assert.equal(isPoseHistoryStable(history.current), true);
});

test("actual movement of the face still resets stable capture", () => {
  const history = { current: [] }, pose = { current: "" };
  for (const x of [100, 300, 500]) {
    rememberPoseMetrics(history, pose, "left", evaluate("left", analysis(), detection({ x })));
  }
  assert.equal(isPoseHistoryStable(history.current), false);
});
