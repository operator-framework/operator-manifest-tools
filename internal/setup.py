# SPDX-License-Identifier: BSD-3-Clause
from setuptools import setup, find_packages

setup(
    name='operator-manifest',
    version='2.0.0',
    long_description=__doc__,
    packages=find_packages(exclude=['tests', 'tests.*']),
    include_package_data=True,
    zip_safe=False,
    url='https://github.com/containerbuildsystem/operator-manifest',
    install_requires=[
        'ruamel.yaml',
        'jsonschema',
    ],
    package_data={'operator_manifest': ['schemas/*.json']},
    classifiers=[
        'License :: OSI Approved :: BSD License',
        'Programming Language :: Python :: 3.6',
        'Programming Language :: Python :: 3.7',
        'Programming Language :: Python :: 3.8',
        'Programming Language :: Python :: 3.9',
    ],
    license="BSD-3-Clause",
    python_requires='>=3.6',
    entry_points={
        'console_scripts': [
            'operator-manifest = operator_manifest.cli:main',
        ],
    },
)
