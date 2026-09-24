import sys
import unittest
from pathlib import Path
from unittest.mock import patch

import numpy as np
from fastapi import HTTPException

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import server


class FacePipelineTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        server.load_verifier_model()

    def test_supplied_model_is_loaded(self):
        self.assertEqual(server.VERIFIER_MODEL['model_type'], 'sface_deep_verifier_128d')
        self.assertEqual(len(server.VERIFIER_MODEL['weights']), 130)

    def test_same_and_unrelated_embeddings(self):
        a, b = np.eye(128)[:2]
        self.assertGreater(server.score_match(a, a)[1], 0.99)
        self.assertLess(server.score_match(a, b)[1], 0.1)

    def test_cosine_threshold_does_not_override_trained_decision(self):
        a, b = np.eye(128)[:2]
        b = 0.28 * a + np.sqrt(1 - 0.28 ** 2) * b
        self.assertGreater(float(a @ b), server.VERIFIER_MODEL['optimal_threshold'])
        self.assertLess(server.score_match(a, b)[1], 0.5)

    def test_blank_image_has_no_faces_and_cannot_enroll(self):
        blank = np.zeros((320, 320, 3), dtype=np.uint8)
        self.assertEqual(server.detect_faces(blank), [])
        with self.assertRaises(HTTPException) as error:
            server.enrollment_vectors(blank)
        self.assertEqual(error.exception.status_code, 400)

    def test_multiple_faces_cannot_enroll(self):
        with patch.object(server, 'detect_faces', return_value=[object(), object()]):
            with self.assertRaises(HTTPException):
                server.enrollment_vectors(np.zeros((320, 320, 3), dtype=np.uint8))

    def test_enrollment_preview_blank_frame(self):
        blank = np.zeros((320, 320, 3), dtype=np.uint8)
        with patch.object(server, 'decode_base64_image', return_value=blank):
            response = server.detect_enrollment_frame(server.FramePayload(image_base64='test'), {})
        self.assertEqual(response, {'faces_detected': 0, 'results': []})

    def test_enrollment_preview_returns_all_boxes_without_recognition(self):
        image = np.zeros((320, 320, 3), dtype=np.uint8)
        faces = np.array([[20, 30, 80, 100], [200, 50, 70, 90]], dtype=float)
        with patch.object(server, 'decode_base64_image', return_value=image), \
             patch.object(server, 'yunet_detector') as detector, \
             patch.object(server, 'sface_recognizer') as recognizer:
            detector.detect.return_value = (None, faces)
            response = server.detect_enrollment_frame(server.FramePayload(image_base64='test'), {})
        self.assertEqual(response['faces_detected'], 2)
        self.assertEqual(response['results'][0]['box'], {'x': 20, 'y': 30, 'width': 80, 'height': 100})
        self.assertEqual(response['results'][1]['box'], {'x': 200, 'y': 50, 'width': 70, 'height': 90})
        recognizer.feature.assert_not_called()

    def test_incompatible_dimensions_are_rejected(self):
        with self.assertRaises(ValueError):
            server.score_match([0] * 64, [0] * 64)

    def test_missing_model_fails_explicitly(self):
        with patch.object(server, 'FACE_VERIFIER_MODEL_PATH', 'nonexistent-test-verifier.json'):
            with self.assertRaises(FileNotFoundError):
                server.load_verifier_model()


if __name__ == '__main__':
    unittest.main()
