import subprocess
import os
import pytest
import shutil
import json
import io
import yaml
import re

AUTHFILE_PATH = os.environ.get('AUTHFILE_PATH')
TEST_DATA_DIR = 'test_data'
DEFAULT_REFERENCES_FILENAME = 'references.json'
DEFAULT_REPLACEMENTS_FILENAME = 'replacements.json'


@pytest.fixture()
def teardown():
    yield
    # clean up space
    if os.path.isfile(DEFAULT_REPLACEMENTS_FILENAME):
        os.remove(DEFAULT_REPLACEMENTS_FILENAME)

    if os.path.isfile(DEFAULT_REFERENCES_FILENAME):
        os.remove(DEFAULT_REFERENCES_FILENAME)


def copy_csv_files_to_manifest_dir(filenames, tmp_path):
    for filename in filenames:
        src_path = os.path.join(TEST_DATA_DIR, filename)
        dest_path = os.path.join(tmp_path, filename)
        shutil.copyfile(src_path, dest_path)


def assert_output_files_are_empty():
    if os.path.exists(DEFAULT_REFERENCES_FILENAME):
        assert os.path.getsize(DEFAULT_REFERENCES_FILENAME) == 0
    if os.path.exists(DEFAULT_REPLACEMENTS_FILENAME):
        assert os.path.getsize(DEFAULT_REPLACEMENTS_FILENAME) == 0


def assert_output_files_have_expected_content(testing_pullspecs):
    with open(DEFAULT_REPLACEMENTS_FILENAME, 'r') as replacements_file:
        actual = json.load(replacements_file)
    expected = {
        pullspec['original']: pullspec['expected'] for pullspec in testing_pullspecs if
        '@sha256:' not in pullspec['original']
    }
    assert len(actual) == len(expected)
    for k, v in expected.items():
        if ':' not in k:
            assert expected[k] == actual[k + ':latest']
        else:
            assert expected[k] == actual[k]

    with open(DEFAULT_REFERENCES_FILENAME, 'r') as references_file:
        references_content = references_file.read()
    for pullspec in testing_pullspecs:
        if ':' not in pullspec['original']:
            assert pullspec['original'] + ':latest' in references_content
        else:
            assert pullspec['original'] in references_content


class TestPinCommand():

    # def test_version(self, teardown, tmp_path):
    #     cmd = subprocess.run(['operator-manifest-tools', 'pinning', 'version'], capture_output=True)
    #     print({"err": cmd.stderr, "output":cmd.stderr})
    #     assert 'asdfasfda' in str(cmd.stderr)

    def test_manifest_dir_empty(self, teardown, tmp_path):
        cmd = subprocess.run(['operator-manifest-tools', 'pinning', 'pin', tmp_path], capture_output=True)
        print({"err": cmd.stderr, "output":cmd.stderr})
        assert 'Missing ClusterServiceVersion in operator manifests' in str(cmd.stderr)
        assert_output_files_are_empty()

    def test_non_existent_manifest_dir(self, teardown):
        cmd = subprocess.run(['operator-manifest-tools', 'pinning', 'pin', 'foo_dir'], capture_output=True)
        assert 'foo_dir is not a directory or does not exist' in str(cmd.stderr)
        assert_output_files_are_empty()

    @pytest.mark.parametrize("csv_filename", [["multiple_csv_1.yaml", "multiple_csv_2.yaml"]])
    def test_multiple_csv(self, csv_filename, teardown, tmp_path):
        copy_csv_files_to_manifest_dir(csv_filename, tmp_path)
        cmd = subprocess.run(['operator-manifest-tools', 'pinning', 'pin', tmp_path], capture_output=True)
        assert 'Operator bundle may contain only 1 CSV file, but contains more' in str(cmd.stderr)
        assert_output_files_are_empty()

    def test_missing_image_property_in_csv(self, teardown, tmp_path):
        copy_csv_files_to_manifest_dir(['missing_image_property_csv.yaml'], tmp_path)
        cmd = subprocess.run(['operator-manifest-tools', 'pinning', 'pin', tmp_path], capture_output=True)
        assert "\\'image\\' is a required property" in str(cmd.stderr)
        assert_output_files_are_empty()

    @pytest.mark.parametrize("csv_filename, testing_pullspecs", [
        ('digest_pinning_csv.yaml',
         [
         {
             'original': 'quay.io/containerbuildsystem/operator-manifest-test-image:v0.9.0',
             'expected': 'quay.io/containerbuildsystem/operator-manifest-test-image@sha256:4b94fbb7acec63ab573ef00ebab577c21f2243e50b1b620f7330a49a393960ef',
             'expected_count': 4,  # defined twice as tag and as digest
             'expected_related_image_names': {'test-operator', 'test-restore-operator'},
         },
         {
             'original': 'quay.io/containerbuildsystem/operator-manifest-test-image',  # implicit latest
             'expected': 'quay.io/containerbuildsystem/operator-manifest-test-image@sha256:1eec301ecce912311d4457ffaf9e7c7bf22d2e8a7e9251ab5a9d105262f69db8',
             'expected_count': 2,
             'expected_related_image_names': {'test-backup-operator'},
         },
         {
             'original': 'quay.io/containerbuildsystem/operator-manifest-test-image@sha256:4b94fbb7acec63ab573ef00ebab577c21f2243e50b1b620f7330a49a393960ef',
             'expected': 'quay.io/containerbuildsystem/operator-manifest-test-image@sha256:4b94fbb7acec63ab573ef00ebab577c21f2243e50b1b620f7330a49a393960ef',
             'expected_count': 4,  # defined twice as tag and as digest
             'expected_related_image_names': {'test-operator', 'test-restore-operator'},
         },
         {
             'original': 'quay.io/containerbuildsystem/operator-manifest-test-image:v0.7.0',
             'expected': 'quay.io/containerbuildsystem/operator-manifest-test-image@sha256:395c1476431ab9af753325a00430e362e84bd419f444b7147a88910c7b13ec70',
             'expected_count': 2,
             'expected_related_image_names': {
                 'operator-manifest-test-image-395c1476431ab9af753325a00430e362e84bd419f444b7147a88910c7b13ec70-annotation'
             },
         },
         {
             'original': 'quay.io/containerbuildsystem/operator-manifest-test-image:v0.2.0',
             'expected': 'quay.io/containerbuildsystem/operator-manifest-test-image@sha256:e2db1a2e64c2d929aa3ac9e95bb6c3cc2083d6912c7d6df1994bf3baffc31fbf',
             'expected_count': 2,
             'expected_related_image_names': {'test_operator'},
         },
         {
             'original': 'quay.io/containerbuildsystem/operator-manifest-test-image:v0.10.0',
             'expected': 'quay.io/containerbuildsystem/operator-manifest-test-image@sha256:7734c909893177f7967ff9b27cd855ab86eeb07f2ad816c1ace8bbdaa335869a',
             'expected_count': 2,
             'expected_related_image_names': {'test-operator-init'},
         },
         {
             'original': 'quay.io/containerbuildsystem/operator-manifest-test-image:v0.3.0',
             'expected': 'quay.io/containerbuildsystem/operator-manifest-test-image@sha256:21514e19c7066b5a643b18074f326cae7c0dedb97acd8c7563d8b42b829e89a9',
             'expected_count': 2,
             'expected_related_image_names': {'test_operator_init'},
         },
         {
             'original': 'quay.io/containerbuildsystem/operator-manifest-test-image:v0.6.0',
             'expected': 'quay.io/containerbuildsystem/operator-manifest-test-image@sha256:1e90a2191d04edf9bc303d505d3c6cccc8078eca1719ed85017a15b73a2df445',
             'expected_count': 2,
             'expected_related_image_names': {
                 'operator-manifest-test-image-1e90a2191d04edf9bc303d505d3c6cccc8078eca1719ed85017a15b73a2df445-annotation'
             },
         },
         {
             'original': 'quay.io/containerbuildsystem/operator-manifest-test-image:v0.5.0',
             'expected': 'quay.io/containerbuildsystem/operator-manifest-test-image@sha256:47c1bbf4cff978bcdcda8d25b818399e123fac6fe129f7804138fd391e091d9c',
             'expected_count': 2,
             'expected_related_image_names': {
                 'operator-manifest-test-image-47c1bbf4cff978bcdcda8d25b818399e123fac6fe129f7804138fd391e091d9c-annotation'
             },
         },
         {
             'original': 'quay.io/containerbuildsystem/operator-manifest-test-image:v0.4.0',
             'expected': 'quay.io/containerbuildsystem/operator-manifest-test-image@sha256:e4492d271fd0730dbf9fce78ce0964de190ec35cc6f6b5f7888cacdbf3c1f1d2',
             'expected_count': 2,
             'expected_related_image_names': {
                 'operator-manifest-test-image-e4492d271fd0730dbf9fce78ce0964de190ec35cc6f6b5f7888cacdbf3c1f1d2-annotation'
             },
         },
         ]),
        pytest.param('digest_pinning_external_repos_csv.yaml', [
        {
            'original': 'quay.io/containerbuildsystem/operator-manifest-test-image:v0.9.0',
            'expected': 'quay.io/containerbuildsystem/operator-manifest-test-image@sha256:4b94fbb7acec63ab573ef00ebab577c21f2243e50b1b620f7330a49a393960ef',
            'expected_count': 2,
            'expected_related_image_names': {'test-operator'},
        },
        {
            'original': 'registry.redhat.io/ubi8/ubi:8.2-265',
            'expected': 'registry.redhat.io/ubi8/ubi@sha256:158e87c4021c4e419b7c127d2d244efe78aa28e0fc99445b0ef405b84b0cf2ef',
            'expected_count': 2,
            'expected_related_image_names': {'test-backup-operator'},
        },
        ],
        marks=pytest.mark.skipif(AUTHFILE_PATH is None, reason="AUTHFILE_PATH was not defined"))
    ])
    def test_digest_pinning(self, csv_filename, testing_pullspecs, teardown, tmp_path):
        copy_csv_files_to_manifest_dir([csv_filename], tmp_path)
        cmd = ['operator-manifest-tools', 'pinning', 'pin', tmp_path]
        if AUTHFILE_PATH:
            cmd += ['--authfile', AUTHFILE_PATH]

        proc = subprocess.run(cmd, capture_output=False)
        proc.check_returncode()

        original_pullspecs = [pullspec['original'] for pullspec in testing_pullspecs]
        expected_pullspecs = [pullspec['expected'] for pullspec in testing_pullspecs]
        csv_content = _get_csv_file_content(csv_filename, tmp_path)

        pullspecs_with_changes = list(
            set(original_pullspecs) - set(original_pullspecs).intersection(set(expected_pullspecs))
        )

        assert_pullspecs_in_csv(csv_content, expected_pullspecs)
        assert_pullspecs_not_in_csv(csv_content, pullspecs_with_changes)

        for pullspec_d in testing_pullspecs:
            count = pullspec_d['expected_count']
            assert_correct_count_of_pullspec_in_csv(csv_content, pullspec_d['expected'],
                                                    expected_count=count)

        pullspec_name_map = {
            tp['expected']: tp['expected_related_image_names']
            for tp in testing_pullspecs
            if 'expected_related_image_names' in tp
        }
        assert_related_images_names(csv_content, pullspec_name_map)
        assert_output_files_have_expected_content(testing_pullspecs)

    @pytest.mark.parametrize('csv_filename, testing_pullspecs', [
        ('related_images_section_specified_csv.yaml', [
            {
                'original': 'quay.io/containerbuildsystem/operator-manifest-test-image:v0.8.0',
                'expected': 'quay.io/containerbuildsystem/operator-manifest-test-image@sha256:f219163f0bdbe36dc50d1c7fdeb0840a5a9bff8ac3922af4ef4c094a88bbf0b3'
            },
            {
                'original': 'quay.io/containerbuildsystem/operator-manifest-test-image:v0.9.0',
                'expected': 'quay.io/containerbuildsystem/operator-manifest-test-image@sha256:4b94fbb7acec63ab573ef00ebab577c21f2243e50b1b620f7330a49a393960ef',
            },
            {
                # quay.io/containerbuildsystem/operator-manifest-test-image
                # cannot test without tag, it will match anything
                'original': None,
                'expected': 'quay.io/containerbuildsystem/operator-manifest-test-image@sha256:1eec301ecce912311d4457ffaf9e7c7bf22d2e8a7e9251ab5a9d105262f69db8'
            },
            {
                'original': None,  # already pinned
                'expected': 'quay.io/containerbuildsystem/operator-manifest-test-image@sha256:4b94fbb7acec63ab573ef00ebab577c21f2243e50b1b620f7330a49a393960ef',
            },
            {
                'original': 'quay.io/containerbuildsystem/operator-manifest-test-image:v0.7.0',
                'expected': 'quay.io/containerbuildsystem/operator-manifest-test-image@sha256:395c1476431ab9af753325a00430e362e84bd419f444b7147a88910c7b13ec70'
            },
            {
                'original': 'quay.io/containerbuildsystem/operator-manifest-test-image:v0.10.0',
                'expected': 'quay.io/containerbuildsystem/operator-manifest-test-image@sha256:7734c909893177f7967ff9b27cd855ab86eeb07f2ad816c1ace8bbdaa335869a',
            },
        ])
    ])
    def test_related_images_section_specified(self, csv_filename, testing_pullspecs, teardown, tmp_path):
        """
        When relatedImages section already exists in csv file, operator-manifest lib
        should skip replacements. Maintainer is responsible for content in this case
        """
        copy_csv_files_to_manifest_dir([csv_filename], tmp_path)
        cmd = subprocess.run(['operator-manifest-tools', 'pinning', 'pin', tmp_path], capture_output=True)
        cmd.check_returncode()
        original_pullspecs = [pullspec['original'] for pullspec in testing_pullspecs if pullspec['original']]
        expected_pullspecs = [pullspec['expected'] for pullspec in testing_pullspecs]
        csv_content = _get_csv_file_content(csv_filename, tmp_path)

        assert_pullspecs_in_csv(csv_content, expected_pullspecs)
        assert_pullspecs_not_in_csv(csv_content, original_pullspecs)

    @pytest.mark.parametrize('csv_filename', ['nonexistent_image_csv.yaml'])
    def test_nonexistent_image(self, csv_filename, teardown, tmp_path):
        copy_csv_files_to_manifest_dir([csv_filename], tmp_path)
        cmd = subprocess.run(['operator-manifest-tools', 'pinning', 'pin', tmp_path], capture_output=True)
        err_msg = "Failed to inspect docker://quay.io/containerbuildsystem/operator-manifest-test-image:nonexistenttag." \
                  " Make sure it exists and is accessible."
        assert err_msg in str(cmd.stderr)

    @pytest.mark.parametrize('csv_filename', ['related_images_defined_on_both_places_csv.yaml'])
    def test_manifests_having_related_images_defined_on_both_places(self, csv_filename, teardown, tmp_path):
        """Change: previously test was supposed to fail. Now it must pass. Replacements themselves are already tested
        in the previous test cases"""
        copy_csv_files_to_manifest_dir([csv_filename], tmp_path)
        cmd = subprocess.run(['operator-manifest-tools', 'pinning', 'pin', tmp_path], capture_output=True)
        cmd.check_returncode()


class TestExtractCommand():

    def test_manifest_dir_empty(self, teardown, tmp_path):
        cmd = subprocess.run(['operator-manifest-tools', 'pinning', 'extract', tmp_path], capture_output=True)
        assert 'Missing ClusterServiceVersion in operator manifests' in str(cmd.stderr)

    def test_non_existent_manifest_dir(self, teardown):
        cmd = subprocess.run(['operator-manifest-tools', 'pinning', 'extract', 'foo_dir'], capture_output=True)
        assert 'foo_dir is not a directory or does not exist' in str(cmd.stderr)

    def test_multiple_csv(self, teardown, tmp_path):
        copy_csv_files_to_manifest_dir(['multiple_csv_1.yaml', 'multiple_csv_2.yaml'], tmp_path)
        cmd = subprocess.run(['operator-manifest-tools', 'pinning', 'extract', tmp_path], capture_output=True)
        assert 'Operator bundle may contain only 1 CSV file, but contains more' in str(cmd.stderr)

    @pytest.mark.parametrize('csv_filename, expected_image_references', [
        ('digest_pinning_csv.yaml', [
            'quay.io/containerbuildsystem/operator-manifest-test-image:latest',
            'quay.io/containerbuildsystem/operator-manifest-test-image:v0.2.0',
            'quay.io/containerbuildsystem/operator-manifest-test-image@sha256:4b94fbb7acec63ab573ef00ebab577c21f2243e50b1b620f7330a49a393960ef',
            'quay.io/containerbuildsystem/operator-manifest-test-image:v0.10.0',
            'quay.io/containerbuildsystem/operator-manifest-test-image:v0.3.0',
            'quay.io/containerbuildsystem/operator-manifest-test-image:v0.9.0',
            'quay.io/containerbuildsystem/operator-manifest-test-image:v0.7.0',
            'quay.io/containerbuildsystem/operator-manifest-test-image:v0.5.0',
            'quay.io/containerbuildsystem/operator-manifest-test-image:v0.4.0',
            'quay.io/containerbuildsystem/operator-manifest-test-image:v0.6.0'
        ])
    ])
    def test_command(self, csv_filename, expected_image_references, teardown, tmp_path):
        copy_csv_files_to_manifest_dir([csv_filename], tmp_path)
        cmd = subprocess.run(['operator-manifest-tools', 'pinning', 'extract', tmp_path], capture_output=True)
        for expected_image_reference in expected_image_references:
            assert expected_image_reference in str(cmd.stdout), f"Expected image reference {expected_image_reference} not extracted"


class TestResolveCommand():

    def test_image_file_does_not_exist(self):
        cmd = subprocess.run(['operator-manifest-tools', 'pinning', 'resolve', 'nonexistentfile'], capture_output=True)
        assert "nonexistentfile is not a directory or does not exist" in str(cmd.stderr)

    @pytest.mark.parametrize('images_file_content,expected_data', [
        ([
            "quay.io/containerbuildsystem/operator-manifest-test-image:latest",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.2.0",
            "quay.io/containerbuildsystem/operator-manifest-test-image@sha256:4b94fbb7acec63ab573ef00ebab577c21f2243e50b1b620f7330a49a393960ef",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.10.0",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.3.0",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.9.0",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.7.0",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.5.0",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.4.0",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.6.0"
        ],
        {
            "quay.io/containerbuildsystem/operator-manifest-test-image:latest": "quay.io/containerbuildsystem/operator-manifest-test-image@sha256:1eec301ecce912311d4457ffaf9e7c7bf22d2e8a7e9251ab5a9d105262f69db8",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.2.0": "quay.io/containerbuildsystem/operator-manifest-test-image@sha256:e2db1a2e64c2d929aa3ac9e95bb6c3cc2083d6912c7d6df1994bf3baffc31fbf",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.10.0": "quay.io/containerbuildsystem/operator-manifest-test-image@sha256:7734c909893177f7967ff9b27cd855ab86eeb07f2ad816c1ace8bbdaa335869a",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.3.0": "quay.io/containerbuildsystem/operator-manifest-test-image@sha256:21514e19c7066b5a643b18074f326cae7c0dedb97acd8c7563d8b42b829e89a9",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.9.0": "quay.io/containerbuildsystem/operator-manifest-test-image@sha256:4b94fbb7acec63ab573ef00ebab577c21f2243e50b1b620f7330a49a393960ef",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.7.0": "quay.io/containerbuildsystem/operator-manifest-test-image@sha256:395c1476431ab9af753325a00430e362e84bd419f444b7147a88910c7b13ec70",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.5.0": "quay.io/containerbuildsystem/operator-manifest-test-image@sha256:47c1bbf4cff978bcdcda8d25b818399e123fac6fe129f7804138fd391e091d9c",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.4.0": "quay.io/containerbuildsystem/operator-manifest-test-image@sha256:e4492d271fd0730dbf9fce78ce0964de190ec35cc6f6b5f7888cacdbf3c1f1d2",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.6.0": "quay.io/containerbuildsystem/operator-manifest-test-image@sha256:1e90a2191d04edf9bc303d505d3c6cccc8078eca1719ed85017a15b73a2df445",
        }
    )])
    def test_command(self, images_file_content, expected_data, tmp_path):
        fp = tmp_path / 'images_file.txt'
        with fp.open('w') as f:
            json.dump(images_file_content, f)
        cmd = subprocess.run(['operator-manifest-tools', 'pinning', 'resolve', fp], capture_output=True)
        actual_data = json.loads(cmd.stdout)
        assert actual_data == expected_data


class TestReplaceCommand():

    @pytest.mark.parametrize('replacements', [
        {
            "quay.io/containerbuildsystem/operator-manifest-test-image": "quay.io/containerbuildsystem/operator-manifest-test-image@sha256:1eec301ecce912311d4457ffaf9e7c7bf22d2e8a7e9251ab5a9d105262f69db8",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.2.0": "quay.io/containerbuildsystem/operator-manifest-test-image@sha256:e2db1a2e64c2d929aa3ac9e95bb6c3cc2083d6912c7d6df1994bf3baffc31fbf",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.10.0": "quay.io/containerbuildsystem/operator-manifest-test-image@sha256:7734c909893177f7967ff9b27cd855ab86eeb07f2ad816c1ace8bbdaa335869a",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.3.0": "quay.io/containerbuildsystem/operator-manifest-test-image@sha256:21514e19c7066b5a643b18074f326cae7c0dedb97acd8c7563d8b42b829e89a9",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.9.0": "quay.io/containerbuildsystem/operator-manifest-test-image@sha256:4b94fbb7acec63ab573ef00ebab577c21f2243e50b1b620f7330a49a393960ef",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.7.0": "quay.io/containerbuildsystem/operator-manifest-test-image@sha256:395c1476431ab9af753325a00430e362e84bd419f444b7147a88910c7b13ec70",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.5.0": "quay.io/containerbuildsystem/operator-manifest-test-image@sha256:47c1bbf4cff978bcdcda8d25b818399e123fac6fe129f7804138fd391e091d9c",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.4.0": "quay.io/containerbuildsystem/operator-manifest-test-image@sha256:e4492d271fd0730dbf9fce78ce0964de190ec35cc6f6b5f7888cacdbf3c1f1d2",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.6.0": "quay.io/containerbuildsystem/operator-manifest-test-image@sha256:1e90a2191d04edf9bc303d505d3c6cccc8078eca1719ed85017a15b73a2df445",
        }
    ])
    def test_non_existent_manifest_dir(self, replacements, teardown, tmp_path):
        fp = tmp_path / DEFAULT_REPLACEMENTS_FILENAME
        with fp.open('w') as f:
            json.dump(replacements, f)
        cmd = subprocess.run(['operator-manifest-tools', 'pinning', 'replace', 'foo_dir', fp], capture_output=True)
        assert 'foo_dir is not a directory or does not exist' in str(cmd.stderr)

    @pytest.mark.parametrize('csv_filename', ['digest_pinning_csv.yaml'])
    def test_non_existent_replacements_file(self, csv_filename, teardown, tmp_path):
        copy_csv_files_to_manifest_dir([csv_filename], tmp_path)
        cmd = subprocess.run(['operator-manifest-tools', 'pinning', 'replace', tmp_path, DEFAULT_REPLACEMENTS_FILENAME], capture_output=True)
        assert f"{DEFAULT_REPLACEMENTS_FILENAME} is not a directory or does not exist" in str(cmd.stderr)

    @pytest.mark.parametrize('csv_filename, replacements', [(
        'digest_pinning_csv.yaml',
        {
            "quay.io/containerbuildsystem/operator-manifest-test-image": "quay.io/containerbuildsystem/operator-manifest-test-image@sha256:1eec301ecce912311d4457ffaf9e7c7bf22d2e8a7e9251ab5a9d105262f69db8",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.2.0": "quay.io/containerbuildsystem/operator-manifest-test-image@sha256:e2db1a2e64c2d929aa3ac9e95bb6c3cc2083d6912c7d6df1994bf3baffc31fbf",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.10.0": "quay.io/containerbuildsystem/operator-manifest-test-image@sha256:7734c909893177f7967ff9b27cd855ab86eeb07f2ad816c1ace8bbdaa335869a",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.3.0": "quay.io/containerbuildsystem/operator-manifest-test-image@sha256:21514e19c7066b5a643b18074f326cae7c0dedb97acd8c7563d8b42b829e89a9",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.9.0": "quay.io/containerbuildsystem/operator-manifest-test-image@sha256:4b94fbb7acec63ab573ef00ebab577c21f2243e50b1b620f7330a49a393960ef",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.7.0": "quay.io/containerbuildsystem/operator-manifest-test-image@sha256:395c1476431ab9af753325a00430e362e84bd419f444b7147a88910c7b13ec70",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.5.0": "quay.io/containerbuildsystem/operator-manifest-test-image@sha256:47c1bbf4cff978bcdcda8d25b818399e123fac6fe129f7804138fd391e091d9c",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.4.0": "quay.io/containerbuildsystem/operator-manifest-test-image@sha256:e4492d271fd0730dbf9fce78ce0964de190ec35cc6f6b5f7888cacdbf3c1f1d2",
            "quay.io/containerbuildsystem/operator-manifest-test-image:v0.6.0": "quay.io/containerbuildsystem/operator-manifest-test-image@sha256:1e90a2191d04edf9bc303d505d3c6cccc8078eca1719ed85017a15b73a2df445",
        }
    )])
    def test_command(self, csv_filename, replacements, teardown, tmp_path):
        print(tmp_path)
        copy_csv_files_to_manifest_dir([csv_filename], tmp_path)
        fp = tmp_path / DEFAULT_REPLACEMENTS_FILENAME
        with fp.open('w') as f:
            json.dump(replacements, f)
        subprocess.run(['operator-manifest-tools', 'pinning', 'replace', '-v', tmp_path, fp], capture_output=True)
        csv_content = _get_csv_file_content(csv_filename, tmp_path)

        assert_pullspecs_in_csv(csv_content, replacements.values())
        assert_pullspecs_not_in_csv(csv_content, replacements.keys())


def _get_csv_file_content(csv_filename, tmp_path):
    csv_file_path = os.path.join(tmp_path, csv_filename)
    with open(csv_file_path, 'r') as f:
        content = f.read()
    return content


def assert_pullspecs_in_csv(csv_content, expected_pullspecs):
    for pullspec in expected_pullspecs:
        if pullspec.endswith('@sha256:'):
            regex = f"\\s{pullspec}"
        else:
            regex = f"\\s{pullspec}\\s"
        assert re.search(regex, csv_content), f"Expected pullspec ({pullspec}) not in csv file."


def assert_pullspecs_not_in_csv(csv_content, forbidden_pullspecs):
    for pullspec in forbidden_pullspecs:
        regex = f"\\s{pullspec}\\s"
        assert not re.search(regex, csv_content), f"Forbidden pullspec ({pullspec}) in csv file."


def assert_correct_count_of_pullspec_in_csv(csv_content, pullspec, expected_count):
    count = csv_content.count(pullspec)
    assert expected_count == count, (
        f"Pullspec {pullspec} found in csv file {count} times, expected {expected_count}"
    )


def assert_related_images_names(csv_content, pullspec_name_map):
    """
        Assert if names in relatedImages section were constructed as expected
        """
    related_images = []
    csv_stream = io.StringIO(csv_content)
    related_images_section = yaml.safe_load(csv_stream)['spec'].get('relatedImages', [])
    related_images.extend(related_images_section)

    def get_names(pullspec_):
        return {
            ri['name']
            for ri in related_images
            if pullspec_ == ri['image']
        }

    all_expected_names = set()
    for pullspec, exp_names in pullspec_name_map.items():
        got_names = get_names(pullspec)
        all_expected_names.update(exp_names)
        assert exp_names == got_names, (
            f"Unexpected names for pullspec {pullspec}: "
            f"expected {exp_names}; got: {got_names}"
        )

    # check if coverage is complete
    all_names = {ri['name'] for ri in related_images}
    assert all_names == all_expected_names, (
        f"Incomplete coverage: all names defined in CSV: "
        f"{all_names} differs from excepted names {all_expected_names}"
    )
