"""Regression coverage for structural outline utilities and palette rejection."""

import unittest

from scripts.check_palette import offenders_in


class PaletteTest(unittest.TestCase):
    def test_outline_offsets_are_spacing(self):
        self.assertEqual(offenders_in(
            'focus-visible:outline-2 focus-visible:outline-offset-4 '
            'outline-offset-12 -outline-offset-2 outline-accent'
        ), [])

    def test_spacing_does_not_hide_unapproved_colors(self):
        self.assertEqual(offenders_in(
            'outline-offset-4 outline-blue-500 bg-cyan-500 text-white'
        ), ['outline-blue-500', 'bg-cyan-500', 'text-white'])


if __name__ == '__main__':
    unittest.main()
