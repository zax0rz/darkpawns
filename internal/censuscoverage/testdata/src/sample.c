/* Fixture oracle for the coverage golden test. */

ACMD(do_quit)
{
  send_to_char("Goodbye, friend.. Come back soon!\r\n", ch);
  sprintf(buf, "Saving %s.\r\n", GET_NAME(ch));
  send_to_char("You have %d gold pieces on hand.\r\n", ch);
  send_to_char("You are hungry.\r\nYou are thirsty.\r\n", ch);
  send_to_char("You have to type quit--no less, to quit!\r\n", ch);
}

ACMD(do_sneak)
{
  act("$n has left the game.", TRUE, ch, 0, 0, TO_ROOM);
}
