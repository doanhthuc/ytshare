import {
  FormProvider,
  type FieldValues,
  type SubmitHandler,
  type UseFormReturn,
} from 'react-hook-form';

interface RHFFormProviderProps<TFieldValues extends FieldValues = FieldValues> {
  form: UseFormReturn<TFieldValues>;
  onSubmit: SubmitHandler<TFieldValues>;
  children: React.ReactNode;
  className?: string;
  id?: string;
}

export function RHFFormProvider<TFieldValues extends FieldValues = FieldValues>({
  form,
  onSubmit,
  children,
  className,
  id,
}: RHFFormProviderProps<TFieldValues>) {
  return (
    <FormProvider {...form}>
      <form id={id} onSubmit={form.handleSubmit(onSubmit)} className={className} noValidate>
        {children}
      </form>
    </FormProvider>
  );
}
