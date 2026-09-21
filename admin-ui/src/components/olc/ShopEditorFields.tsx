import type { OlcPatchOperation, OlcSchema, OlcSchemaField, OlcShop } from '../../api/olc';
import { ServerProposal } from './ServerProposal';
import { VNumPicker } from './VNumPicker';

interface ShopEditorFieldsProps {
  schema: OlcSchema;
  shop: OlcShop;
  dirty: string[];
  disabled?: boolean;
  onOperation: (operations: OlcPatchOperation[]) => void;
}

const MESSAGE_OPERATIONS: Record<string, string> = {
  message_no_item_keeper: 'set_no_item1',
  message_no_item_player: 'set_no_item2',
  message_no_buy: 'set_no_buy',
  message_no_cash_keeper: 'set_no_cash1',
  message_no_cash_player: 'set_no_cash2',
  message_buy: 'set_buy_message',
  message_sell: 'set_sell_message',
};

const MESSAGE_INDEX: Record<string, number> = {
  message_no_item_keeper: 0,
  message_no_item_player: 1,
  message_no_buy: 2,
  message_no_cash_keeper: 3,
  message_no_cash_player: 4,
  message_buy: 5,
  message_sell: 6,
};

function fieldLabel(field: OlcSchemaField, changed: boolean) {
  return (
    <div className="mb-2 flex items-baseline justify-between gap-3">
      <label htmlFor={`shop-${field.key}`} className="text-sm font-semibold text-ink">{field.label}</label>
      {changed && <span className="font-mono text-[10px] uppercase tracking-wider text-accent">changed</span>}
    </div>
  );
}

function messageText(shop: OlcShop, index: number): string {
  const message = shop.messages[index] || '';
  return message.startsWith('%s ') ? message.slice(3) : message;
}

function scalarValue(shop: OlcShop, field: OlcSchemaField): string | number {
  switch (field.key) {
    case 'buy_profit': return shop.buyProfit;
    case 'sell_profit': return shop.sellProfit;
    case 'keeper': return shop.keeperVnum;
    case 'open_hour_1': return shop.openHour1;
    case 'open_hour_2': return shop.openHour2;
    case 'close_hour_1': return shop.closeHour1;
    case 'close_hour_2': return shop.closeHour2;
    case 'message_no_item_keeper': return messageText(shop, 0);
    case 'message_no_item_player': return messageText(shop, 1);
    case 'message_no_buy': return messageText(shop, 2);
    case 'message_no_cash_keeper': return messageText(shop, 3);
    case 'message_no_cash_player': return messageText(shop, 4);
    case 'message_buy': return messageText(shop, 5);
    case 'message_sell': return messageText(shop, 6);
    default: return '';
  }
}

function isSet(value: number, bit: number): boolean {
  return (value & 2 ** bit) !== 0;
}

function withBit(value: number, bit: number, enabled: boolean): number {
  return enabled ? value | 2 ** bit : value & ~(2 ** bit);
}

function AddNumberForm({
  name,
  label,
  operation,
  disabled,
  onOperation,
}: {
  name: string;
  label: string;
  operation: string;
  disabled: boolean;
  onOperation: (operations: OlcPatchOperation[]) => void;
}) {
  return (
    <form
      key={name}
      className="flex gap-2"
      onSubmit={(event) => {
        event.preventDefault();
        const form = event.currentTarget;
        const value = Number(new FormData(form).get('value'));
        if (Number.isFinite(value)) onOperation([{ kind: operation, value }]);
        form.reset();
      }}
    >
      <input name="value" type="number" aria-label={label} disabled={disabled} className="min-w-0 flex-1 border border-rule bg-paper px-3 py-2 text-sm text-ink" />
      <button type="submit" disabled={disabled} className="border border-rule px-3 py-2 text-xs font-semibold uppercase tracking-wider text-ink hover:bg-ink hover:text-paper disabled:cursor-not-allowed disabled:opacity-50">Add</button>
    </form>
  );
}

function NumberList({
  label,
  values,
  operation,
  removeOperation,
  disabled,
  onOperation,
}: {
  label: string;
  values: number[];
  operation: string;
  removeOperation: string;
  disabled: boolean;
  onOperation: (operations: OlcPatchOperation[]) => void;
}) {
  return (
    <section className="border-t border-rule pt-5">
      <h3 className="mb-3 text-lg text-ink">{label}</h3>
      <div className="space-y-2">
        {values.map((value, index) => (
          <div key={`${value}-${index}`} className="flex items-center justify-between border border-rule bg-paper px-3 py-2">
            <span className="font-mono text-sm text-ink">{value}</span>
            <button type="button" disabled={disabled} onClick={() => onOperation([{ kind: removeOperation, index }])} className="text-xs font-semibold uppercase tracking-wider text-accent hover:text-accent-deep disabled:opacity-50">Remove</button>
          </div>
        ))}
        <AddNumberForm name={label} label={`Add ${label}`} operation={operation} disabled={disabled} onOperation={onOperation} />
      </div>
    </section>
  );
}

export function ShopEditorFields({ schema, shop, dirty, disabled = false, onOperation }: ShopEditorFieldsProps) {
  const changed = dirty.includes('shop');
  const fields = schema.fields.filter((field) => !['products', 'rooms', 'trade_namelist'].includes(field.key));

  return (
    <div className="space-y-5">
      <section className="border-t border-rule pt-5">
        <h2 className="mb-4 text-lg text-ink">Shop fields</h2>
        <div className="grid gap-5 lg:grid-cols-2">
          {fields.map((field) => {
            if (field.control === 'checkbox_group') {
              const value = field.key === 'flags' ? shop.flags : shop.withWho;
              const operation = field.key === 'flags' ? 'set_flags' : 'set_no_trade';
              return (
                <fieldset key={field.key} className="lg:col-span-2">
                  {fieldLabel(field, changed)}
                  <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
                    {(field.options || []).map((option) => (
                      <label key={option.value} className="flex min-h-10 items-center gap-2 border border-rule bg-paper px-3 py-2 text-sm text-ink hover:bg-paper-deep">
                        <input
                          type="checkbox"
                          checked={isSet(value, option.value)}
                          disabled={disabled}
                          onChange={(event) => onOperation([{
                            kind: operation,
                            ...(field.key === 'flags'
                              ? { value: withBit(value, option.value, event.currentTarget.checked) }
                              : { bit: option.value, enabled: event.currentTarget.checked }),
                          }])}
                          className="h-4 w-4 accent-accent"
                        />
                        <span>{option.label}</span>
                      </label>
                    ))}
                  </div>
                </fieldset>
              );
            }
            const value = scalarValue(shop, field);
            const isMessage = Boolean(MESSAGE_OPERATIONS[field.key]);
            const operation = isMessage
              ? MESSAGE_OPERATIONS[field.key]
              : field.key === 'buy_profit' ? 'set_buy_profit'
                : field.key === 'sell_profit' ? 'set_sell_profit'
                  : field.key === 'keeper' ? 'set_keeper'
                    : field.key === 'open_hour_1' ? 'set_open_hour1'
                      : field.key === 'open_hour_2' ? 'set_open_hour2'
                        : field.key === 'close_hour_1' ? 'set_close_hour1'
                          : field.key === 'close_hour_2' ? 'set_close_hour2'
                            : `set_${field.key}`;
            return (
              <div key={field.key}>
                {fieldLabel(field, changed)}
                {field.key === 'keeper' ? (
                  <VNumPicker
                    kind="mob"
                    value={shop.keeperVnum}
                    identity="shop-keeper"
                    label={field.label}
                    showLabel={false}
                    disabled={disabled}
                    min={-1}
                    onCommit={(raw) => onOperation([{ kind: operation, value: Number(raw) }])}
                  />
                ) : (
                  <ServerProposal
                    value={value}
                    identity={`shop-${field.key}`}
                    disabled={disabled}
                    multiline={false}
                    min={field.bounds?.min}
                    max={field.bounds?.max}
                    onCommit={(raw) => onOperation([{ kind: operation, ...(isMessage ? { text: raw, index: MESSAGE_INDEX[field.key] } : field.key.includes('profit') ? { float: Number(raw) } : { value: Number(raw) }) }])}
                  />
                )}
                {field.bounds && <p className="mt-1 font-mono text-[11px] text-ink-muted">{field.bounds.min}–{field.bounds.max}</p>}
              </div>
            );
          })}
        </div>
      </section>

      <NumberList label="Products" values={shop.products} operation="add_product" removeOperation="remove_product" disabled={disabled} onOperation={onOperation} />
      <NumberList label="Shop rooms" values={shop.rooms} operation="add_room" removeOperation="remove_room" disabled={disabled} onOperation={onOperation} />

      <section className="border-t border-rule pt-5">
        <h3 className="mb-3 text-lg text-ink">Trade namelist</h3>
        <div className="space-y-2">
          {shop.buyWords.map((word, index) => (
            <div key={`${word}-${index}`} className="flex items-center justify-between border border-rule bg-paper px-3 py-2">
              <span className="text-sm text-ink"><span className="font-mono text-ink-muted">{shop.buyTypes[index] ?? 0}</span> {word}</span>
              <button type="button" disabled={disabled} onClick={() => onOperation([{ kind: 'remove_namelist', index }])} className="text-xs font-semibold uppercase tracking-wider text-accent hover:text-accent-deep disabled:opacity-50">Remove</button>
            </div>
          ))}
          <form
            key={`namelist-${shop.buyWords.length}`}
            className="grid gap-2 sm:grid-cols-[7rem_1fr_auto]"
            onSubmit={(event) => {
              event.preventDefault();
              const form = event.currentTarget;
              const data = new FormData(form);
              const type = Number(data.get('type'));
              const word = String(data.get('word') || '');
              if (Number.isFinite(type) && word) onOperation([{ kind: 'add_namelist', value: type, text: word }]);
              form.reset();
            }}
          >
            <input name="type" type="number" aria-label="Trade type number" disabled={disabled} placeholder="Type" className="border border-rule bg-paper px-3 py-2 text-sm text-ink" />
            <input name="word" type="text" aria-label="Trade namelist word" disabled={disabled} placeholder="Namelist word" className="border border-rule bg-paper px-3 py-2 text-sm text-ink" />
            <button type="submit" disabled={disabled} className="border border-rule px-3 py-2 text-xs font-semibold uppercase tracking-wider text-ink hover:bg-ink hover:text-paper disabled:cursor-not-allowed disabled:opacity-50">Add</button>
          </form>
        </div>
      </section>
    </div>
  );
}
