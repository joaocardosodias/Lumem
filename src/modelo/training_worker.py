import os

from ml_service.application.training_service import TrainingJobStore, TrainingService


def main() -> None:
    store = TrainingJobStore()
    service = TrainingService(store, auto_start=False, recover_interrupted=True)
    poll_seconds = float(os.getenv('TRAINING_WORKER_POLL_SECONDS', '1'))
    print('worker de treinamento iniciado', flush=True)
    service.run_forever(poll_seconds)


if __name__ == '__main__':
    main()
