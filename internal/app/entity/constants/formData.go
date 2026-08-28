package constants

// Form-data row types (entity.Variable.Type for RestBodyFormData).
const (
	FormDataTypeText = "text"
	FormDataTypeFile = "file"
)

func NormalizeFormDataType(t string) string {
	if t == FormDataTypeFile {
		return FormDataTypeFile
	}
	return FormDataTypeText
}
