#!/usr/bin/python3

from setuptools import setup


with open('../README.md') as f:
    readme = f.read()

setup(
    name='penlog',
    version='0.1.0',
    description='PENLog provides a specification, library, and tooling for simple machine readable logging',
    long_description=readme,
    long_description_content_type='text/markdown',
    license='Apache2',
    author='Stefan Tatschner and Tobias Specht',
    author_email='stefan.tatschner@aisec.fraunhofer.de',
    url='https://github.com/Fraunhofer-AISEC/penlog',
    package_data={
        'penlog': ['py.typed'],
    },
    py_modules=['penlog'],
)

