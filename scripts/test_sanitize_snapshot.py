import copy
import importlib.util
import pathlib
import unittest


SCRIPT = pathlib.Path(__file__).with_name("sanitize-snapshot.py")
SPEC = importlib.util.spec_from_file_location("sanitize_snapshot", SCRIPT)
SANITIZER = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
SPEC.loader.exec_module(SANITIZER)


class SanitizeSnapshotTest(unittest.TestCase):
    def test_uuid_replacement_is_case_insensitive_and_explicitly_synthetic(self):
        lower = "123e4567-e89b-12d3-a456-426614174000"
        upper = lower.upper()

        lower_result = SANITIZER.sanitize_string(lower)
        upper_result = SANITIZER.sanitize_string(upper)

        self.assertEqual(lower_result, upper_result)
        self.assertRegex(lower_result, r"^demo-uuid-[0-9a-f]{24}$")
        self.assertNotIn(lower, lower_result)
        self.assertNotIn(upper, upper_result)

    def test_kubernetes_uids_are_unique_referential_and_idempotent(self):
        shared_uid = "123E4567-E89B-12D3-A456-426614174000"
        snapshot = {
            "kubernetes": {
                "context": "raw",
                "nodes": [
                    {"apiVersion": "v1", "kind": "Node", "name": "raw-node-a", "uid": shared_uid},
                    {"apiVersion": "v1", "kind": "Node", "name": "raw-node-b", "uid": shared_uid},
                ],
                "workloads": [
                    {"apiVersion": "apps/v1", "kind": "Deployment", "namespace": "app", "name": "api", "uid": shared_uid}
                ],
                "pods": [
                    {
                        "apiVersion": "v1",
                        "kind": "Pod",
                        "namespace": "app",
                        "name": "api-abcde",
                        "uid": shared_uid,
                        "ownerReferences": [
                            {"apiVersion": "apps/v1", "kind": "Deployment", "name": "api", "uid": shared_uid}
                        ],
                    }
                ],
                "services": [],
                "endpointSlices": [],
                "runningImages": [
                    {
                        "image": "example.invalid/api:v1",
                        "workloads": [
                            {"apiVersion": "apps/v1", "kind": "Deployment", "namespace": "app", "name": "api", "uid": shared_uid}
                        ],
                    }
                ],
            }
        }

        first = SANITIZER.sanitize_snapshot(copy.deepcopy(snapshot))
        second = SANITIZER.sanitize_snapshot(copy.deepcopy(first))

        self.assertEqual(first, second)
        objects = first["kubernetes"]["nodes"] + first["kubernetes"]["workloads"] + first["kubernetes"]["pods"]
        self.assertEqual(len(objects), len({item["uid"] for item in objects}))
        owner = first["kubernetes"]["pods"][0]["ownerReferences"][0]
        self.assertEqual(owner["uid"], first["kubernetes"]["workloads"][0]["uid"])
        workload_ref = first["kubernetes"]["runningImages"][0]["workloads"][0]
        self.assertEqual(workload_ref["uid"], first["kubernetes"]["workloads"][0]["uid"])
        self.assertTrue(all(item["uid"].startswith("demo-uid-") for item in objects))


if __name__ == "__main__":
    unittest.main()
