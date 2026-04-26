import { Controller, type FieldPath, type FieldValues, useFormContext } from 'react-hook-form';

import { Input, type InputProps, Label } from '@/components/ui';
import { cn } from '@/shared/utils';

type RHFInputProps<TFieldValues extends FieldValues = FieldValues> = Omit<InputProps, 'name'> & {
  name: FieldPath<TFieldValues>;
  label?: string;
  description?: string;
};

export function RHFInput<TFieldValues extends FieldValues = FieldValues>({
  name,
  label,
  description,
  className,
  ...inputProps
}: RHFInputProps<TFieldValues>) {
  const { control } = useFormContext<TFieldValues>();
  return (
    <Controller
      control={control}
      name={name}
      render={({ field, fieldState }) => (
        <div className={cn('flex w-full flex-col gap-2', className)}>
          {label ? (
            <Label htmlFor={name} className={cn(fieldState.invalid && 'text-danger')}>
              {label}
            </Label>
          ) : null}
          <Input
            id={name}
            aria-invalid={fieldState.invalid || undefined}
            {...field}
            value={field.value ?? ''}
            {...inputProps}
          />
          {description ? <p className="text-xs text-muted-foreground">{description}</p> : null}
          {fieldState.error?.message ? (
            <p className="text-xs text-danger">{fieldState.error.message}</p>
          ) : null}
        </div>
      )}
    />
  );
}
