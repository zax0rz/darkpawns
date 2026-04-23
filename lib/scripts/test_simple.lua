function test()
  log("ch = " .. tostring(ch))
  if ch then
    log("ch.level = " .. tostring(ch.level))
  end
  return true
end