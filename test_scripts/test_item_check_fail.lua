-- Test: item_check returns false when shop does not buy item type
local result = item_check(obj)
if result then
    test_result = "pass"
else
    test_result = "fail"
end
