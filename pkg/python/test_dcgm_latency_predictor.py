import os
import unittest

import dcgm_latency_predictor as predictor


class PredictorTest(unittest.TestCase):
    def setUp(self):
        csv_path = os.path.join(
            os.path.dirname(os.path.abspath(__file__)),
            "results.csv",
        )
        self.baselines = predictor.load_baselines(csv_path)

    def test_load_standard_model_name(self):
        self.assertIn("vgg19_bs128_224x224", self.baselines)
        self.assertAlmostEqual(self.baselines["vgg19_bs128_224x224"], 133.1972, places=4)

    def test_normalize_metric_supports_percent(self):
        self.assertAlmostEqual(predictor.normalize_metric(67), 0.67, places=6)
        self.assertAlmostEqual(predictor.normalize_metric(0.67), 0.67, places=6)

    def test_high_pressure_is_slower_than_low_pressure(self):
        low_pressure = predictor.predict_latency_ms(
            "resnet50_bs64_224x224",
            self.baselines,
            sm_active=22,
            sm_occupancy=28,
            dram_active=30,
        )
        high_pressure = predictor.predict_latency_ms(
            "resnet50_bs64_224x224",
            self.baselines,
            sm_active=82,
            sm_occupancy=78,
            dram_active=58,
        )
        self.assertGreater(high_pressure["predicted_latency_ms"], low_pressure["predicted_latency_ms"])


if __name__ == "__main__":
    unittest.main()
