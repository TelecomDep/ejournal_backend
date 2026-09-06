import unittest

from report_common import short_person_name, student_display_label


class ReportCommonTest(unittest.TestCase):
    def test_short_person_name(self):
        self.assertEqual(short_person_name("Добромилов Артём Александрович"), "Добромилов А.А.")
        self.assertEqual(short_person_name("  Петров   Пётр   Сергеевич "), "Петров П.С.")
        self.assertEqual(short_person_name("student-a8f42"), "student-a8f42")

    def test_student_reference_is_not_shortened_without_consent(self):
        student = {
            "student_label": "Добромилов Артём Александрович",
            "student_ref": "anonymous student reference",
            "personal_data_consent": {"accepted": False},
        }
        self.assertEqual(student_display_label(student), "anonymous student reference")


if __name__ == "__main__":
    unittest.main()
