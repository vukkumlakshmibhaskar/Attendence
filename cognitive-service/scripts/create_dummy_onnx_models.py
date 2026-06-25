from __future__ import annotations

import argparse
from pathlib import Path

import onnx
from onnx import TensorProto, helper, numpy_helper
import numpy as np


def create_constant_classifier(path: Path, probabilities: list[float]) -> None:
    input_tensor = helper.make_tensor_value_info(
        "image_uint8", TensorProto.UINT8, [1, 3, 64, 64]
    )
    output_tensor = helper.make_tensor_value_info(
        "probabilities", TensorProto.FLOAT, [1, len(probabilities)]
    )
    constant = numpy_helper.from_array(
        np.asarray([probabilities], dtype=np.float32), name="constant_probs"
    )
    node = helper.make_node("Constant", inputs=[], outputs=["probabilities"], value=constant)
    graph = helper.make_graph(
        [node],
        path.stem,
        [input_tensor],
        [output_tensor],
    )
    model = helper.make_model(
        graph,
        producer_name="classroom-attendance-dummy-models",
        opset_imports=[helper.make_opsetid("", 13)],
    )
    model.ir_version = 8
    onnx.checker.check_model(model)
    onnx.save(model, path)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--output-dir", default="models")
    args = parser.parse_args()

    output_dir = Path(args.output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)

    create_constant_classifier(output_dir / "emotion_int8_dummy.onnx", [0.72, 0.18, 0.10])
    create_constant_classifier(output_dir / "gaze_int8_dummy.onnx", [0.81, 0.19])
    create_constant_classifier(output_dir / "liveness_int8_dummy.onnx", [0.08, 0.92])

    print(f"Dummy ONNX models written to {output_dir.resolve()}")


if __name__ == "__main__":
    main()
