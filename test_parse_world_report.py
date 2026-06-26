import parse_world
import parse_world_fixed


def test_generate_report_handles_empty_parse_results(tmp_path):
    for parser_module in (parse_world, parse_world_fixed):
        parser = parser_module.WorldParser(str(tmp_path))

        report = parser.generate_report()

        assert "Total rooms parsed: 0" in report
        assert "Rooms with exits: 0 (0.0%)" in report
        assert "Average exits per room: 0.00" in report
