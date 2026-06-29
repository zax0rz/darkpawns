"""Pytest configuration for Dark Pawns E2E web tests."""

import os
import pytest


@pytest.fixture(scope="session")
def base_url() -> str:
    """Base URL for the Dark Pawns server under test.

    Defaults to http://localhost:8080. Override with the DARKPAWNS_BASE_URL
    environment variable (e.g. ``DARKPAWNS_BASE_URL=http://192.168.1.120:8080``
    for a remote instance).
    """
    return os.environ.get("DARKPAWNS_BASE_URL", "http://localhost:8080")
