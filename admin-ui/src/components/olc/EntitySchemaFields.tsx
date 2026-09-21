import type {
  OlcMob,
  OlcObject,
  OlcPatchOperation,
  OlcSchema,
  OlcSchemaField,
} from '../../api/olc';
import { ServerProposal } from './ServerProposal';

type Entity = OlcMob | OlcObject;

interface EntitySchemaFieldsProps {
  schema: OlcSchema;
  kind: 'mob' | 'obj';
  entity: Entity;
  dirty: string[];
  disabled?: boolean;
  onOperation: (operations: OlcPatchOperation[]) => void;
}

const MOB_OPERATIONS: Record<string, string> = {
  keywords: 'set_keywords',
  short_description: 'set_short_description',
  long_description: 'set_long_description',
  detailed_description: 'set_detailed_description',
  noise: 'set_noise',
  sex: 'set_sex',
  hitroll: 'set_hitroll',
  damroll: 'set_damroll',
  damage_dice: 'set_damage_dice',
  damage_sides: 'set_damage_sides',
  hp_dice: 'set_hp_dice',
  hp_sides: 'set_hp_sides',
  hp_plus: 'set_hp_plus',
  ac: 'set_ac',
  exp: 'set_exp',
  gold: 'set_gold',
  position: 'set_position',
  default_position: 'set_default_position',
  attack: 'set_attack',
  level: 'set_level',
  alignment: 'set_alignment',
  race: 'set_race',
  action_flags: 'set_action_flag',
  affect_flags: 'set_affect_flag',
};

const OBJECT_OPERATIONS: Record<string, string> = {
  keywords: 'set_keywords',
  short_description: 'set_short_description',
  long_description: 'set_long_description',
  action_description: 'set_action_description',
  type: 'set_type',
  extra_flags: 'set_extra_flag',
  wear_flags: 'set_wear_flag',
  weight: 'set_weight',
  cost: 'set_cost',
  cost_per_day: 'set_cost_per_day',
};

function isMob(entity: Entity): entity is OlcMob {
  return 'actionFlags' in entity;
}

function fieldValue(field: OlcSchemaField, entity: Entity): string | number {
  if (isMob(entity)) {
    switch (field.key) {
      case 'keywords': return entity.keywords;
      case 'short_description': return entity.shortDesc;
      case 'long_description': return entity.longDesc;
      case 'detailed_description': return entity.detailedDesc;
      case 'noise': return entity.noise;
      case 'sex': return entity.sex;
      case 'hitroll': return 20 - entity.thac0;
      case 'damroll': return entity.damage.plus;
      case 'damage_dice': return entity.damage.num;
      case 'damage_sides': return entity.damage.sides;
      case 'hp_dice': return entity.hp.num;
      case 'hp_sides': return entity.hp.sides;
      case 'hp_plus': return entity.hp.plus;
      case 'ac': return entity.ac;
      case 'exp': return entity.exp;
      case 'gold': return entity.gold;
      case 'position': return entity.position;
      case 'default_position': return entity.defaultPos;
      case 'attack': return entity.bareHandAttack;
      case 'level': return entity.level;
      case 'alignment': return entity.alignment;
      case 'race': return entity.race;
      default: return '';
    }
  }

  switch (field.key) {
    case 'keywords': return entity.keywords;
    case 'short_description': return entity.shortDesc;
    case 'long_description': return entity.longDesc;
    case 'action_description': return entity.actionDesc;
    case 'type': return entity.typeFlag;
    case 'weight': return entity.weight;
    case 'cost': return entity.cost;
    case 'cost_per_day': return entity.loadPercent;
    default: return '';
  }
}

function flagIsSet(entity: Entity, field: OlcSchemaField, bit: number, label: string, storage?: string): boolean {
  if (isMob(entity)) {
    const flags = field.key === 'action_flags' ? entity.actionFlags : entity.affectFlags;
    return flags.includes(storage || label);
  }
  const words = field.key === 'extra_flags' ? entity.extraFlags : entity.wearFlags;
  const word = Math.floor(bit / 32);
  return (words[word] & 2 ** (bit % 32)) !== 0;
}

function FieldLabel({ field, dirty }: { field: OlcSchemaField; dirty: boolean }) {
  return (
    <div className="mb-2 flex items-baseline justify-between gap-3">
      <label htmlFor={`olc-${field.key}`} className="text-sm font-semibold text-ink">{field.label}</label>
      {dirty && <span className="font-mono text-[10px] uppercase tracking-wider text-accent">changed</span>}
    </div>
  );
}

function operationFor(kind: 'mob' | 'obj', field: OlcSchemaField): string | undefined {
  return (kind === 'mob' ? MOB_OPERATIONS : OBJECT_OPERATIONS)[field.key];
}

function ScalarField({
  field,
  kind,
  entity,
  dirty,
  disabled,
  onOperation,
}: {
  field: OlcSchemaField;
  kind: 'mob' | 'obj';
  entity: Entity;
  dirty: boolean;
  disabled: boolean;
  onOperation: (operations: OlcPatchOperation[]) => void;
}) {
  const operation = operationFor(kind, field);
  if (!operation) return null;
  const current = fieldValue(field, entity);
  const commit = (raw: string) => {
    if (field.key === 'cost_per_day') {
      onOperation([{ kind: operation, float: Number(raw) }]);
    } else if (field.control === 'number' || field.control === 'select') {
      onOperation([{ kind: operation, value: Number(raw) }]);
    } else {
      onOperation([{ kind: operation, text: raw }]);
    }
  };

  return (
    <div>
      <FieldLabel field={field} dirty={dirty} />
      <ServerProposal
        value={current}
        identity={`olc-${field.key}`}
        disabled={disabled}
        multiline={field.control === 'textarea'}
        min={field.bounds?.min}
        max={field.bounds?.max}
        onCommit={commit}
      />
      {field.bounds && <p className="mt-1 font-mono text-[11px] text-ink-muted">{field.bounds.min}–{field.bounds.max}</p>}
    </div>
  );
}

function FlagField({
  field,
  entity,
  disabled,
  dirty,
  onOperation,
}: {
  field: OlcSchemaField;
  entity: Entity;
  disabled: boolean;
  dirty: boolean;
  onOperation: (operations: OlcPatchOperation[]) => void;
}) {
  const operation = field.key === 'action_flags' ? 'set_action_flag' : field.key === 'affect_flags' ? 'set_affect_flag' : field.key === 'extra_flags' ? 'set_extra_flag' : 'set_wear_flag';
  return (
    <fieldset className="lg:col-span-2">
      <FieldLabel field={field} dirty={dirty} />
      <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
        {(field.options || []).map((option) => (
          <label key={option.value} className="flex min-h-10 items-center gap-2 border border-rule bg-paper px-3 py-2 text-sm text-ink hover:bg-paper-deep">
            <input
              type="checkbox"
              checked={flagIsSet(entity, field, option.value, option.label, option.storage)}
              disabled={disabled}
              onChange={(event) => onOperation([{ kind: operation, bit: option.value, enabled: event.currentTarget.checked }])}
              className="h-4 w-4 accent-accent"
            />
            <span>{option.label}</span>
          </label>
        ))}
      </div>
    </fieldset>
  );
}

export function EntitySchemaFields({ schema, kind, entity, dirty, disabled = false, onOperation }: EntitySchemaFieldsProps) {
  return (
    <section className="border-t border-rule pt-5">
      <h2 className="mb-4 text-lg text-ink">{kind === 'mob' ? 'Mob fields' : 'Object fields'}</h2>
      <div className="grid gap-5 lg:grid-cols-2">
        {schema.fields.map((field) => {
          if (field.key === 'script_name' || field.key === 'script_flags' || field.key === 'values') return null;
          if (field.control === 'checkbox_group') {
            return <FlagField key={field.key} field={field} entity={entity} dirty={dirty.includes(field.key)} disabled={disabled} onOperation={onOperation} />;
          }
          if (field.control === 'select') {
            const current = fieldValue(field, entity);
            const operation = operationFor(kind, field);
            if (!operation) return null;
            return (
              <div key={field.key}>
                <FieldLabel field={field} dirty={dirty.includes(field.key)} />
                <select
                  id={`olc-${field.key}`}
                  value={String(current)}
                  disabled={disabled}
                  onChange={(event) => onOperation([{ kind: operation, value: Number(event.currentTarget.value) }])}
                  className="w-full border border-rule bg-paper px-3 py-2 text-sm text-ink outline-none focus:border-accent focus:ring-1 focus:ring-accent disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {(field.options || []).map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
                </select>
              </div>
            );
          }
          return <ScalarField key={field.key} field={field} kind={kind} entity={entity} dirty={dirty.includes(field.key)} disabled={disabled} onOperation={onOperation} />;
        })}
      </div>
    </section>
  );
}
