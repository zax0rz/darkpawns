/* A deliberately tiny oracle stand-in for the golden test. */

ACMD(do_quit)
{
  send_to_char("Goodbye, friend.. Come back soon!\r\n", ch);
  act("$n has left the game.", TRUE, ch, 0, 0, TO_ROOM);
  sprintf(buf, "Saving %s.\r\n", GET_NAME(ch));
  stc("You have to type quit--no less, to quit!\r\n", ch);
  send_to_char("Return to the temple and QUIT to leave the game and keep"
               " your equipment.\r\n", ch);
}
